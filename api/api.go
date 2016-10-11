// Package api provides value types and convenience functions around them for
// the secret-volume service.
package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// A SecretSource determines and denotes which secrets.Provider implementation
// a volume should use.
type SecretSource int

const (
	// Unknown source. Volumes with unknown sources will not be handled.
	Unknown SecretSource = iota
	// Talos source. Volumes with the Talos source will be handled by
	// https://github.com/spotify/talos.
	Talos
)

func (s SecretSource) String() string {
	switch s {
	case Talos:
		return "Talos"
	default:
		return "Unknown"
	}
}

// MarshalJSON returns a string representation of a SecretSource.
func (s SecretSource) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", s)), nil
}

// UnmarshalJSON unmarshals a SecretSource from its string representation.
func (s *SecretSource) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return errors.Wrapf(err, "cannot unmarshal %v", s)
	}

	switch strings.ToLower(str) {
	case "talos":
		*s = Talos
	default:
		*s = Unknown
	}

	return nil
}

// PEM represents PEM encoded data. It is a string (rather than []byte) to
// prevent the JSON encoder further encoding it as a base64 string.
type PEM string

// A KeyPair contains PEM encoded data for a Certificate and a PrivateKey.
type KeyPair struct {
	Certificate PEM
	PrivateKey  PEM
}

// NewKeyPair returns a new KeyPair by reading PEM data from the supplied cert
// and key files
func NewKeyPair(cert, key string) (KeyPair, error) {
	certPEM, err := ioutil.ReadFile(cert)
	if err != nil {
		return KeyPair{}, errors.Wrapf(err, "cannot read %v", cert)
	}
	keyPEM, err := ioutil.ReadFile(key)
	if err != nil {
		return KeyPair{}, errors.Wrapf(err, "cannot read %v", key)
	}
	return KeyPair{PEM(certPEM), PEM(keyPEM)}, nil
}

// ToCertificate builds a tls.Certificate from KeyPair PEM data.
func (k KeyPair) ToCertificate() (tls.Certificate, error) {
	crt, err := tls.X509KeyPair([]byte(k.Certificate), []byte(k.PrivateKey))
	return crt, errors.Wrap(err, "cannot parse keypair")
}

// A Volume represents a 'secret volume' in which secrets for a particular
// resource (i.e. a Docker container) will be stored.
type Volume struct {
	// ID is a unique identifier for the volume.
	ID string
	// Source determines which secrets.Provider will provide secrets for this
	// volume.
	Source SecretSource
	// Tags may be passed to the secrets.Provider to request or filter specific
	// secrets.
	Tags url.Values
	// The KeyPair is used for secrets.Providers that require authentication.
	KeyPair KeyPair `json:"-"`
}

// WriteJSON writes a JSON representation of a Volume to the supplied io.Writer.
func (v *Volume) WriteJSON(w io.Writer) error {
	return errors.Wrapf(json.NewEncoder(w).Encode(v), "cannot write JSON for %v", v)
}

// ReadVolumeJSON creates a Volume by reading its JSON representation from the
// supplied io.Reader.
func ReadVolumeJSON(r io.Reader) (*Volume, error) {
	v := &Volume{}
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return nil, errors.Wrap(err, "cannot read JSON")
	}
	return v, nil
}

// A volumeCreation represents the JSON required to create a Volume, including
// the KeyPair.
type volumeCreation struct {
	ID      string
	Source  SecretSource
	Tags    url.Values
	KeyPair KeyPair
}

// ReadVolumeJSONWithKeyPair is a variant of ReadVolumeFromJSON that includes
// the KeyPair. KeyPairs are only relevant at volume creation time, after which
// they are not persisted.
func ReadVolumeJSONWithKeyPair(r io.Reader) (*Volume, error) {
	v := &volumeCreation{}
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return nil, errors.Wrap(err, "cannot read JSON")
	}
	return &Volume{ID: v.ID, Source: v.Source, Tags: v.Tags, KeyPair: v.KeyPair}, nil
}

// Volumes represents a slice of Volumes.
type Volumes []*Volume

// WriteJSON writes a JSON representation of Volumes to the supplied io.Writer.
func (vs Volumes) WriteJSON(w io.Writer) error {
	return errors.Wrapf(json.NewEncoder(w).Encode(vs), "cannot write JSON for %v", vs)
}

// ReadVolumesJSON creates Volumes by reading their JSON representation from the
// supplied io.Reader.
func ReadVolumesJSON(r io.Reader) (Volumes, error) {
	v := &Volumes{}
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return nil, errors.Wrap(err, "cannot read JSON")
	}
	return *v, nil
}

func (v *Volume) String() string {
	return fmt.Sprintf("Volume id=%v source=%v, tags=%v, keypair=%+v", v.ID, v.Source, v.Tags, v.KeyPair)
}

// Secrets represents a set of secret files produced by a secrets.Producer. It
// provides a similar API to the stdlib tar package, with Next() returning a
// SecretsHeader for the next file or io.EOF when no files remain.
type Secrets interface {
	// Volume returns the Volume these Secrets were produced for.
	Volume() *Volume
	// Next advances to the next secrets file or directory.
	Next() (*SecretsHeader, error)
	// Read reads from the current secrets file, returning 0, io.EOF when that
	// file has been consumed. Call Next to advance to the next secrets file.
	Read([]byte) (int, error)
	// Close closes any resources consumed by these Secrets.
	Close() error
}

// A SecretsHeader contains information about an individual secret file.
type SecretsHeader struct {
	Path     string
	FileInfo os.FileInfo
}
