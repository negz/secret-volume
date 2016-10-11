// Package volume manages the creation, deletion, and inspection of secret
// volumes.
package volume

import (
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/uber-go/zap"

	"github.com/negz/secret-volume/api"
	"github.com/negz/secret-volume/secrets"
)

// ErrExists is returned when when attempting to create a volume whose mount
// point exists. Note this does not mean the volume already exists, just that a
// conflicting path exists.
type ErrExists string

func (e ErrExists) Error() string {
	return string(e)
}

// ErrNonExist is returned when attempting to get or destroy a volume that does
// not exist.
type ErrNonExist string

func (e ErrNonExist) Error() string {
	return string(e)
}

// NotFound signals that this error should return a HTTP 404 not found if it
// causes a HTTP request to fail.
func (e ErrNonExist) NotFound() bool {
	return true
}

// A Manager manages CRD operations for secret volumes.
type Manager interface {
	// Create mounts and populates the requested secret volume.
	Create(v *api.Volume) error
	// Destroy destroys the secret volume specified by id.
	Destroy(id string) error
	// Gets returns secret volumes by their id.
	Get(id string) (*api.Volume, error)
	// List lists all extant secret volumes.
	List() (api.Volumes, error)
	// MetadataFile returns the metadata filename. Each api.Volume is encoded as
	// JSON in a metadata file at the root of its mountpoint.
	MetadataFile() string
}

type manager struct {
	m           Mounter
	fs          afero.Fs
	af          *afero.Afero
	producerFor secrets.Producers
	meta        string
	dmode       os.FileMode
	fmode       os.FileMode
}

// A ManagerOption represents an argument to NewManager.
type ManagerOption func(*manager) error

// Filesystem allows a Manager to be backed by any filesystem
// implementation supported by https://github.com/spf13/afero. The OS filesystem
// is used by default.
func Filesystem(fs afero.Fs) ManagerOption {
	return func(sm *manager) error {
		sm.fs = fs
		sm.af = &afero.Afero{Fs: fs}
		return nil
	}
}

// MetadataFile specifies an alternative metadata filename in which to store
// JSON encoded representations of each api.Volume at their root directory. It
// defaults to '.meta'.
func MetadataFile(f string) ManagerOption {
	return func(sm *manager) error {
		sm.meta = f
		return nil
	}
}

// DirMode specifies the octal mode with which to create directories beneath the
// root of a secret volume. It defaults to 0700.
func DirMode(m os.FileMode) ManagerOption {
	return func(sm *manager) error {
		sm.dmode = m
		return nil
	}
}

// FileMode specifies the octal mode with which to create files in a secret
// volume. It defaults to 0600.
func FileMode(m os.FileMode) ManagerOption {
	return func(sm *manager) error {
		sm.fmode = m
		return nil
	}
}

// NewManager creates a new Manager backed by the provided secret producers.
func NewManager(m Mounter, sp secrets.Producers, mo ...ManagerOption) (Manager, error) {
	fs := afero.NewOsFs()
	sm := &manager{
		m,
		fs,
		&afero.Afero{Fs: fs},
		sp,
		".meta",
		0700,
		0600,
	}
	for _, o := range mo {
		if err := o(sm); err != nil {
			return nil, errors.Wrap(err, "cannot apply manager option")
		}
	}
	return sm, nil
}

func (sm *manager) createFile(id, file string) (afero.File, error) {
	p := path.Join(sm.m.Path(id), file)
	d := path.Dir(p)
	// Talos serves tarballs without directories.
	if exists, err := sm.af.DirExists(d); err != nil {
		return nil, errors.Wrap(err, "cannot test directory existence while creating file")
	} else if !exists {
		log.Debug("creating directory", zap.String("path", d), zap.String("type", "implicit"))
		if err := sm.af.MkdirAll(d, sm.dmode); err != nil {
			return nil, errors.Wrap(err, "cannot create parent directories while creating file")
		}
	}
	log.Debug("creating file", zap.String("path", p))
	m := os.O_CREATE | os.O_EXCL | os.O_WRONLY
	f, err := sm.fs.OpenFile(p, m, sm.fmode)
	return f, errors.Wrap(err, "cannot open file for creation")
}

func (sm *manager) writeSecrets(v *api.Volume, s api.Secrets) error {
	for {
		h, err := s.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "cannot iterate to next secret file")
		}

		if h.FileInfo.IsDir() {
			d := path.Join(sm.m.Path(v.ID), h.Path)
			log.Debug("creating directory", zap.String("path", d), zap.String("type", "explicit"))
			if err := sm.fs.MkdirAll(d, sm.dmode); err != nil {
				return errors.Wrap(err, "cannot create secret directory")
			}
		} else {
			f, err := sm.createFile(v.ID, h.Path)
			if err != nil {
				return errors.Wrap(err, "cannot create secret file")
			}
			if _, err := io.Copy(f, s); err != nil {
				f.Close()
				return errors.Wrapf(err, "cannot copy secret to file %v", f.Name())
			}
			if err := f.Close(); err != nil {
				return errors.Wrapf(err, "cannot close secret file %v", f.Name())
			}
		}
	}
}

func (sm *manager) writeMetadata(v *api.Volume) error {
	f, err := sm.createFile(v.ID, sm.meta)
	if err != nil {
		return errors.Wrap(err, "cannot create metadata file")
	}
	defer f.Close()
	return errors.Wrap(v.WriteJSON(f), "cannot write to metadata file")
}

func (sm *manager) Create(v *api.Volume) error {
	log.Debug("creating volume", zap.String("id", v.ID))

	if exists, err := sm.af.Exists(sm.m.Path(v.ID)); err != nil {
		return errors.Wrap(err, "cannot test volume path existence")
	} else if exists {
		return ErrExists("volume exists")
	}
	sp, exists := sm.producerFor[v.Source]
	if !exists {
		return errors.New("no producer for secret type")
	}
	s, err := sp.For(v)
	if err != nil {
		return errors.Wrap(err, "cannot produce secret")
	}
	defer s.Close()
	if err := sm.fs.MkdirAll(sm.m.Path(v.ID), sm.dmode); err != nil {
		return errors.Wrap(err, "cannot create volume path")
	}
	if err := sm.m.Mount(v); err != nil {
		return errors.Wrap(err, "cannot mount volume")
	}
	if err := sm.writeSecrets(v, s); err != nil {
		return errors.Wrap(err, "cannot write secrets")
	}
	if err := sm.writeMetadata(v); err != nil {
		return errors.Wrap(err, "cannot write metadata")
	}
	log.Info("created volume", zap.String("id", v.ID), zap.String("path", sm.m.Path(v.ID)))
	return nil
}

func (sm *manager) Destroy(id string) error {
	log.Debug("destroying volume", zap.String("id", id))

	if exists, err := sm.af.DirExists(sm.m.Path(id)); err != nil {
		return errors.Wrap(err, "cannot test volume path existence")
	} else if !exists {
		return ErrNonExist("volume not found")
	}
	if err := sm.m.Unmount(id); err != nil {
		return errors.Wrap(err, "cannot unmount volume")
	}
	if err := sm.fs.RemoveAll(sm.m.Path(id)); err != nil {
		return errors.Wrap(err, "cannot remove volume path")
	}

	log.Info("destroyed volume", zap.String("id", id), zap.String("path", sm.m.Path(id)))
	return nil
}

func (sm *manager) readMetadata(id string) (*api.Volume, error) {
	f, err := sm.fs.Open(path.Join(sm.m.Path(id), sm.MetadataFile()))
	if err != nil {
		return nil, errors.Wrap(err, "cannot open metadata file")
	}
	defer f.Close()

	v, err := api.ReadVolumeJSON(f)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read from metadata file")
	}

	log.Debug("read metadata", zap.String("id", v.ID))
	return v, nil
}

func (sm *manager) Get(id string) (*api.Volume, error) {
	log.Debug("getting volume", zap.String("id", id))

	if exists, err := sm.af.DirExists(sm.m.Path(id)); err != nil {
		return nil, errors.Wrap(err, "cannot test volume path existence")
	} else if !exists {
		return nil, ErrNonExist("volume not found")
	}
	return sm.readMetadata(id)
}

func (sm *manager) List() (api.Volumes, error) {
	log.Debug("listing volumes")

	if exists, err := sm.af.DirExists(sm.m.Root()); err != nil {
		return nil, errors.Wrap(err, "cannot test parent directory existence")
	} else if !exists {
		return nil, errors.New("parent directory does not exist")
	}

	f, err := sm.fs.Open(sm.m.Root())
	if err != nil {
		return nil, errors.Wrap(err, "cannot open parent directory for listing")
	}

	dirs, err := f.Readdirnames(0)
	if err != nil {
		return nil, errors.Wrap(err, "cannot list volumes in parent directory")
	}

	vols := make([]*api.Volume, 0, len(dirs))
	for _, id := range dirs {
		v, err := sm.readMetadata(id)
		if err != nil {
			// TODO(negz): Metric-i-fy this.
			log.Debug("unparseable volume", zap.Error(err))
			continue
		}
		vols = append(vols, v)
	}
	return vols, nil
}

func (sm *manager) MetadataFile() string {
	return sm.meta
}
