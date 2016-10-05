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

// NewTarGz creates an api.Secrets backed by the supplied io.ReadCloser, which
// is expected to be a gzipped tarball of secret files.
func NewTarGz(v *api.Volume, r io.ReadCloser) (api.Secrets, error) {
	z, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &tarGz{v, r, z, tar.NewReader(z)}, nil
}

// OpenTarGz creates an api.Secrets backed by the supplied file, which is
// expected to be a gzipped tarball of secret files.
func OpenTarGz(v *api.Volume, fs afero.Fs, file string) (api.Secrets, error) {
	z, err := fs.Open(file)
	if err != nil {
		return nil, err
	}
	s, err := NewTarGz(v, z)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Volume returns the Volume these secrets were produced for.
func (sd *tarGz) Volume() *api.Volume {
	return sd.v
}

func fromTarHeader(h *tar.Header) *api.SecretsHeader {
	return &api.SecretsHeader{Path: h.Name, FileInfo: h.FileInfo()}
}

// Next advances to the next secrets file or directory.
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

// Read reads from the current secrets file, returning 0, io.EOF when that file
// has been consumed. Call Next to advance to the next secrets file.
func (sd *tarGz) Read(b []byte) (int, error) {
	return sd.t.Read(b)
}

// Close closes the underlying gzip reader and tar readers.
func (sd *tarGz) Close() error {
	if err := sd.z.Close(); err != nil {
		return err
	}
	return sd.r.Close()
}
