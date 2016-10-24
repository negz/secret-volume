package secrets

import (
	"io"
	"os"
	"time"

	"github.com/negz/secret-volume/api"
)

type singleSecretFileInfo struct {
	name string
}

func (s *singleSecretFileInfo) Name() string {
	return s.name
}

func (s *singleSecretFileInfo) Size() int64 {
	return 0
}

func (s *singleSecretFileInfo) Mode() os.FileMode {
	return 0
}

func (s *singleSecretFileInfo) ModTime() time.Time {
	return time.Now()
}

func (s *singleSecretFileInfo) IsDir() bool {
	return false
}

func (s *singleSecretFileInfo) Sys() interface{} {
	return nil
}

type singleFile struct {
	v    *api.Volume
	name string
	r    io.Reader
	t    api.SecretType
	read bool
}

// NewSingleFile returns a set of Secrets containing only a single file, the
// provided io.Reader.
func NewSingleFile(v *api.Volume, n string, r io.Reader, t api.SecretType) api.Secrets {
	return &singleFile{v: v, name: n, r: r, t: t, read: false}
}

func (s *singleFile) Volume() *api.Volume {
	return s.v
}

func (s *singleFile) Next() (*api.SecretsHeader, error) {
	if s.read {
		return nil, io.EOF
	}
	s.read = true
	return &api.SecretsHeader{
		Path:     s.name,
		Type:     s.t,
		FileInfo: &singleSecretFileInfo{name: s.name},
	}, nil
}

func (s *singleFile) Read(b []byte) (int, error) {
	return s.r.Read(b)
}

func (s *singleFile) Close() error {
	return nil
}
