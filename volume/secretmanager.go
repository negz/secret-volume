package volume

import (
	"io"
	"os"
	"path"

	"github.com/spf13/afero"
	"github.com/uber-go/zap"

	"github.com/negz/secret-volume/api"
	"github.com/negz/secret-volume/secrets"
)

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
			return nil, err
		}
	}
	return sm, nil
}

func (sm *manager) createFile(id, file string) (afero.File, error) {
	p := path.Join(sm.m.Path(id), file)
	d := path.Dir(p)
	// Talos serves tarballs without directories.
	if exists, err := sm.af.DirExists(d); err != nil {
		return nil, err
	} else if !exists {
		log.Debug("creating directory", zap.String("path", d), zap.String("type", "implicit"))
		if err := sm.af.MkdirAll(d, sm.dmode); err != nil {
			return nil, err
		}
	}
	log.Debug("creating file", zap.String("path", p))
	return sm.fs.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, sm.fmode)
}

func (sm *manager) writeSecrets(v *api.Volume, s api.Secrets) error {
	for {
		h, err := s.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if h.FileInfo.IsDir() {
			d := path.Join(sm.m.Path(v.ID), h.Path)
			log.Debug("creating directory", zap.String("path", d), zap.String("type", "explicit"))
			if err := sm.fs.MkdirAll(d, sm.dmode); err != nil {
				return err
			}
		} else {
			f, err := sm.createFile(v.ID, h.Path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, s); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}
}

func (sm *manager) writeMetadata(v *api.Volume) error {
	// TODO(negz): Use some binary serialisation for the metadata?
	f, err := sm.createFile(v.ID, sm.meta)
	if err != nil {
		return err
	}
	defer f.Close()
	return v.WriteJSON(f)
}

func (sm *manager) Create(v *api.Volume) error {
	if exists, err := sm.af.Exists(sm.m.Path(v.ID)); err != nil {
		return err
	} else if exists {
		return PathExistsError
	}
	sp, exists := sm.producerFor[v.Source]
	if !exists {
		return secrets.UnhandledSecretSourceError
	}
	s, err := sp.For(v)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := sm.fs.MkdirAll(sm.m.Path(v.ID), sm.dmode); err != nil {
		return err
	}
	if err := sm.m.Mount(v); err != nil {
		return err
	}
	if err := sm.writeSecrets(v, s); err != nil {
		return err
	}
	if err := sm.writeMetadata(v); err != nil {
		return err
	}
	log.Info("created volume", zap.String("id", v.ID), zap.String("path", sm.m.Path(v.ID)))
	return nil
}

func (sm *manager) Destroy(id string) error {
	if exists, err := sm.af.DirExists(sm.m.Path(id)); err != nil {
		return err
	} else if !exists {
		return PathDoesNotExistError
	}
	if err := sm.m.Unmount(id); err != nil {
		return err
	}
	if err := sm.fs.RemoveAll(sm.m.Path(id)); err != nil {
		return err
	}

	log.Info("destroyed volume", zap.String("id", id), zap.String("path", sm.m.Path(id)))
	return nil
}

func (sm *manager) readMetadata(id string) (*api.Volume, error) {
	f, err := sm.fs.Open(path.Join(sm.m.Path(id), sm.MetadataFile()))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	v, err := api.ReadVolumeJSON(f)
	if err != nil {
		return nil, err
	}

	log.Debug("read metadata", zap.String("id", v.ID))
	return v, nil
}

func (sm *manager) Get(id string) (*api.Volume, error) {
	if exists, err := sm.af.DirExists(sm.m.Path(id)); err != nil {
		return nil, err
	} else if !exists {
		return nil, UnknownVolumeError
	}
	return sm.readMetadata(id)
}

func (sm *manager) List() (api.Volumes, error) {
	if exists, err := sm.af.DirExists(sm.m.Root()); err != nil {
		return nil, err
	} else if !exists {
		return nil, MissingMountpointError
	}

	f, err := sm.fs.Open(sm.m.Root())
	if err != nil {
		return nil, err
	}

	dirs, err := f.Readdirnames(0)
	if err != nil {
		return nil, err
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
