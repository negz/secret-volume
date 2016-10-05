package secrets

import (
	"archive/tar"
	"compress/gzip"
	"io"

	"github.com/spf13/afero"
	"github.com/uber-go/zap"

	"github.com/negz/secret-volume/api"
)

type tarGz struct {
	v *api.Volume
	r io.ReadCloser
	z *gzip.Reader
	t *tar.Reader
}

func NewTarGz(v *api.Volume, r io.ReadCloser) (api.Secrets, error) {
	z, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &tarGz{v, r, z, tar.NewReader(z)}, nil
}

func OpenTarGz(v *api.Volume, fs afero.Fs, f string) (api.Secrets, error) {
	z, err := fs.Open(f)
	if err != nil {
		return nil, err
	}
	s, err := NewTarGz(v, z)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (sd *tarGz) Volume() *api.Volume {
	return sd.v
}

func fromTarHeader(h *tar.Header) *api.SecretsHeader {
	return &api.SecretsHeader{Path: h.Name, FileInfo: h.FileInfo()}
}

func (sd *tarGz) Next() (*api.SecretsHeader, error) {
	for {
		h, err := sd.t.Next()
		if err != nil {
			return nil, err
		}
		if !(h.FileInfo().Mode().IsDir() || h.FileInfo().Mode().IsRegular()) {
			log.Debug("ignoring strange file",
				zap.String("path", h.Name), zap.Uint("filemode", uint(h.FileInfo().Mode())))
			continue
		}
		log.Debug("found file", zap.String("path", h.Name))
		return fromTarHeader(h), nil
	}
}

func (sd *tarGz) Read(b []byte) (int, error) {
	return sd.t.Read(b)
}

func (sd *tarGz) Close() error {
	if err := sd.z.Close(); err != nil {
		return err
	}
	return sd.r.Close()
}