package secrets

import (
	"archive/tar"
	"compress/gzip"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/uber-go/zap"

	"github.com/negz/secret-volume/api"
)

type tarGz struct {
	v *api.Volume
	r io.ReadCloser
	z *gzip.Reader
	t *tar.Reader
	s api.SecretType
}

type TarGzOption func(*tarGz) error

func TarGzSecretType(s api.SecretType) TarGzOption {
	return func(t *tarGz) error {
		t.s = s
		return nil
	}
}

// NewTarGz creates an api.Secrets backed by the supplied io.ReadCloser, which
// is expected to be a gzipped tarball of secret files.
func NewTarGz(v *api.Volume, r io.ReadCloser, tgzo ...TarGzOption) (api.Secrets, error) {
	z, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "cannot build new gzip reader")
	}
	t := &tarGz{v, r, z, tar.NewReader(z), api.UnknownSecretType}
	for _, o := range tgzo {
		if err := o(t); err != nil {
			return nil, errors.Wrap(err, "cannot apply tar.gz secrets option")
		}
	}
	return t, nil
}

// OpenTarGz creates an api.Secrets backed by the supplied file, which is
// expected to be a gzipped tarball of secret files.
func OpenTarGz(v *api.Volume, fs afero.Fs, file string, tgzo ...TarGzOption) (api.Secrets, error) {
	z, err := fs.Open(file)
	if err != nil {
		return nil, errors.Wrap(err, "cannot open tar.gz secrets")
	}
	s, err := NewTarGz(v, z, tgzo...)
	return s, errors.Wrap(err, "cannot build tar.gz secrets")
}

// Volume returns the Volume these secrets were produced for.
func (sd *tarGz) Volume() *api.Volume {
	return sd.v
}

func (sd *tarGz) fromTarHeader(h *tar.Header) *api.SecretsHeader {
	return &api.SecretsHeader{Path: h.Name, Type: sd.s, FileInfo: h.FileInfo()}
}

// Next advances to the next secrets file or directory.
func (sd *tarGz) Next() (*api.SecretsHeader, error) {
	for {
		h, err := sd.t.Next()
		if err == io.EOF {
			return nil, err
		}
		if err != nil {
			return nil, errors.Wrap(err, "cannot iterate to next file in tarball")
		}
		if !(h.FileInfo().Mode().IsDir() || h.FileInfo().Mode().IsRegular()) {
			log.Debug("ignoring strange file",
				zap.String("path", h.Name), zap.Uint("filemode", uint(h.FileInfo().Mode())))
			continue
		}
		log.Debug("found file", zap.String("path", h.Name))
		return sd.fromTarHeader(h), nil
	}
}

// Read reads from the current secrets file, returning 0, io.EOF when that file
// has been consumed. Call Next to advance to the next secrets file.
func (sd *tarGz) Read(b []byte) (int, error) {
	i, err := sd.t.Read(b)
	if err == io.EOF {
		return i, err
	}
	return i, errors.Wrap(err, "cannot read from tarball")
}

// Close closes the underlying gzip reader and tar readers.
func (sd *tarGz) Close() error {
	if err := sd.z.Close(); err != nil {
		return errors.Wrap(err, "cannot close tarball reader")
	}
	return errors.Wrap(sd.r.Close(), "cannot close gzip reader")
}
