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

type secretVolumeManager struct {
	m           Mounter
	fs          afero.Fs
	af          *afero.Afero
	producerFor secrets.SecretProducers
	meta        string
	dmode       os.FileMode
	fmode       os.FileMode
}

type SecretVolumeManagerOption func(*secretVolumeManager) error

func Filesystem(fs afero.Fs) SecretVolumeManagerOption {
	return func(vm *secretVolumeManager) error {
		vm.fs = fs
		vm.af = &afero.Afero{fs}
		return nil
	}
}

func MetadataFile(f string) SecretVolumeManagerOption {
	return func(vm *secretVolumeManager) error {
		vm.meta = f
		return nil
	}
}

func DirMode(m os.FileMode) SecretVolumeManagerOption {
	return func(vm *secretVolumeManager) error {
		vm.dmode = m
		return nil
	}
}

func FileMode(m os.FileMode) SecretVolumeManagerOption {
	return func(vm *secretVolumeManager) error {
		vm.fmode = m
		return nil
	}
}

func NewSecretVolumeManager(m Mounter, sp secrets.SecretProducers, vmo ...SecretVolumeManagerOption) (VolumeManager, error) {
	fs := afero.NewOsFs()
	vm := &secretVolumeManager{
		m,
		fs,
		&afero.Afero{fs},
		sp,
		".meta",
		0700,
		0600,
	}
	for _, o := range vmo {
		if err := o(vm); err != nil {
			return nil, err
		}
	}
	return vm, nil
}

func (vm *secretVolumeManager) createFile(id, file string) (afero.File, error) {
	p := path.Join(vm.m.Path(id), file)
	d := path.Dir(p)
	// Talos serves tarballs without directories.
	if exists, err := vm.af.DirExists(d); err != nil {
		return nil, err
	} else if !exists {
		log.Debug("creating directory", zap.String("path", d), zap.String("type", "implicit"))
		if err := vm.af.MkdirAll(d, vm.dmode); err != nil {
			return nil, err
		}
	}
	log.Debug("creating file", zap.String("path", p))
	return vm.fs.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, vm.fmode)
}

func (vm *secretVolumeManager) writeSecrets(v *api.Volume, s api.Secrets) error {
	for {
		h, err := s.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if h.FileInfo.IsDir() {
			d := path.Join(vm.m.Path(v.Id), h.Path)
			log.Debug("creating directory", zap.String("path", d), zap.String("type", "explicit"))
			if err := vm.fs.MkdirAll(d, vm.dmode); err != nil {
				return err
			}
		} else {
			f, err := vm.createFile(v.Id, h.Path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, s); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
}

func (vm *secretVolumeManager) writeMetadata(v *api.Volume) error {
	// TODO(negz): Use some binary serialisation for the metadata?
	f, err := vm.createFile(v.Id, vm.meta)
	if err != nil {
		return err
	}
	defer f.Close()
	return v.WriteJSON(f)
}

func (vm *secretVolumeManager) Create(v *api.Volume) error {
	if exists, err := vm.af.Exists(vm.m.Path(v.Id)); err != nil {
		return err
	} else if exists {
		return PathExistsError
	}
	sp, exists := vm.producerFor[v.Source]
	if !exists {
		return secrets.UnhandledSecretSourceError
	}
	s, err := sp.For(v)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := vm.fs.MkdirAll(vm.m.Path(v.Id), vm.dmode); err != nil {
		return err
	}
	if err := vm.m.Mount(v); err != nil {
		return err
	}
	if err := vm.writeSecrets(v, s); err != nil {
		return err
	}
	if err := vm.writeMetadata(v); err != nil {
		return err
	}
	log.Info("created volume", zap.String("id", v.Id), zap.String("path", vm.m.Path(v.Id)))
	return nil
}

func (vm *secretVolumeManager) Destroy(id string) error {
	if exists, err := vm.af.DirExists(vm.m.Path(id)); err != nil {
		return err
	} else if !exists {
		return PathDoesNotExistError
	}
	if err := vm.m.Unmount(id); err != nil {
		return err
	}
	if err := vm.fs.RemoveAll(vm.m.Path(id)); err != nil {
		return err
	}

	log.Info("destroyed volume", zap.String("id", id), zap.String("path", vm.m.Path(id)))
	return nil
}

func (vm *secretVolumeManager) readMetadata(id string) (*api.Volume, error) {
	f, err := vm.fs.Open(path.Join(vm.m.Path(id), vm.MetadataFile()))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	v, err := api.ReadVolumeJSON(f)
	if err != nil {
		return nil, err
	}

	log.Debug("read metadata", zap.String("id", v.Id))
	return v, nil
}

func (vm *secretVolumeManager) Get(id string) (*api.Volume, error) {
	if exists, err := vm.af.DirExists(vm.m.Path(id)); err != nil {
		return nil, err
	} else if !exists {
		return nil, UnknownVolumeError
	}
	return vm.readMetadata(id)
}

func (vm *secretVolumeManager) List() (api.Volumes, error) {
	if exists, err := vm.af.DirExists(vm.m.Root()); err != nil {
		return nil, err
	} else if !exists {
		return nil, MissingMountpointError
	}

	f, err := vm.fs.Open(vm.m.Root())
	if err != nil {
		return nil, err
	}

	dirs, err := f.Readdirnames(0)
	if err != nil {
		return nil, err
	}

	vols := make([]*api.Volume, 0, len(dirs))
	for _, id := range dirs {
		v, err := vm.readMetadata(id)
		if err != nil {
			// TODO(negz): Metric-i-fy this.
			log.Debug("unparseable volume", zap.Error(err))
			continue
		}
		vols = append(vols, v)
	}
	return vols, nil
}

func (vm *secretVolumeManager) MetadataFile() string {
	return vm.meta
}
