// Package fixtures provides convenience test fixtures shared by other
// packages of secret-volume.
package fixtures

import (
	"io"
	"net/url"
	"os"
	"time"

	"github.com/negz/secret-volume/api"
	"github.com/pkg/errors"
)

// TestVolumeWithCert generates a volume fixture from the supplied PEM certs
func TestVolumeWithCert(c, k string) (*api.Volume, error) {
	kp, err := api.NewKeyPair(c, k)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create test volume keypair")
	}
	// Validate the keypair is parseable early
	if _, err := kp.ToCertificate(); err != nil {
		return nil, errors.Wrap(err, "cannot parse test volume cert")
	}
	v := &api.Volume{
		ID:      "hash",
		Source:  api.Talos,
		Tags:    url.Values{"tag": []string{"awesome"}},
		KeyPair: kp,
	}
	return v, nil
}

// TestVolume is a volume fixture
var TestVolume = &api.Volume{
	ID:      "hash",
	Source:  api.Talos,
	Tags:    url.Values{"tag": []string{"awesome"}},
	KeyPair: api.KeyPair{},
}

// TestVolumes is a slice of volume fixtures
var TestVolumes = api.Volumes{TestVolume}

type boringFileInfo struct{}

func (f *boringFileInfo) Name() string {
	return "derp"
}

func (f *boringFileInfo) Size() int64 {
	return 0
}

func (f *boringFileInfo) Mode() os.FileMode {
	return 0
}

func (f *boringFileInfo) ModTime() time.Time {
	return time.Now()
}

func (f *boringFileInfo) IsDir() bool {
	return false
}

func (f *boringFileInfo) Sys() interface{} {
	return nil
}

type boringSecrets struct {
	v    *api.Volume
	read bool
}

// NewBoringSecrets returns a very boring secrets fixture
func NewBoringSecrets(v *api.Volume) api.Secrets {
	return &boringSecrets{v, false}
}

func (s *boringSecrets) Volume() *api.Volume {
	return s.v
}

func (s *boringSecrets) Next() (*api.SecretsHeader, error) {
	if s.read {
		return nil, io.EOF
	}
	s.read = true
	return &api.SecretsHeader{Path: "womp", FileInfo: &boringFileInfo{}}, nil
}

func (s *boringSecrets) Read(b []byte) (int, error) {
	return 0, io.EOF
}

func (s *boringSecrets) Close() error {
	return nil
}
