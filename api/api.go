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
)

type SecretSource int

const (
	Unknown SecretSource = iota
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

func (s SecretSource) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", s)), nil
}

func (s *SecretSource) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	switch strings.ToLower(str) {
	case "talos":
		*s = Talos
	default:
		*s = Unknown
	}

	return nil
}

type PEM []byte

type KeyPair struct {
	Certificate PEM
	PrivateKey  PEM
}

func NewKeyPair(cert, key string) (KeyPair, error) {
	certPEM, err := ioutil.ReadFile(cert)
	if err != nil {
		return KeyPair{}, err
	}
	keyPEM, err := ioutil.ReadFile(key)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{certPEM, keyPEM}, nil
}

func (k KeyPair) ToCertificate() (tls.Certificate, error) {
	return tls.X509KeyPair(k.Certificate, k.PrivateKey)
}

type Volume struct {
	ID      string
	Source  SecretSource
	Tags    url.Values
	KeyPair KeyPair `json:"-"`
}

func (v *Volume) WriteJSON(w io.Writer) error {
	return json.NewEncoder(w).Encode(v)
}

func ReadVolumeJSON(r io.Reader) (*Volume, error) {
	v := &Volume{}
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return nil, err
	}
	return v, nil
}

type Volumes []*Volume

func (vs Volumes) WriteJSON(w io.Writer) error {
	return json.NewEncoder(w).Encode(vs)
}

func ReadVolumesJSON(r io.Reader) (Volumes, error) {
	v := &Volumes{}
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return nil, err
	}
	return *v, nil
}

func (v *Volume) String() string {
	return fmt.Sprintf("Volume id=%v source=%v, tags=%v, keypair=%+v", v.ID, v.Source, v.Tags, v.KeyPair)
}

type Secrets interface {
	Volume() *Volume
	Next() (*SecretsHeader, error)
	Read([]byte) (int, error)
	Close() error
}

type SecretsHeader struct {
	Path     string
	FileInfo os.FileInfo
}
