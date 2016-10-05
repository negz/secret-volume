package volume

import (
	"fmt"
	"hash/fnv"
	"io"
	"path"
	"reflect"
	"testing"

	"github.com/spf13/afero"

	"github.com/negz/secret-volume/api"
	"github.com/negz/secret-volume/fixtures"
	"github.com/negz/secret-volume/secrets"
)

type boringSecretProducer struct {
	s api.Secrets
}

func (sp *boringSecretProducer) For(v *api.Volume) (api.Secrets, error) {
	return sp.s, nil
}

var secretManagerTests = []struct {
	v *api.Volume
	f string
}{
	{fixtures.TestVolume, ""},
	{fixtures.TestVolume, "../fixtures/yaml.tar.gz"},
}

func TestSecretManager(t *testing.T) {
	m := NewNoopMounter("/noop")
	fs := afero.NewMemMapFs()

	for _, tt := range secretManagerTests {
		var s api.Secrets
		if tt.f == "" {
			s = fixtures.NewBoringSecrets(tt.v)
		} else {
			tgz, err := secrets.OpenTarGzSecrets(tt.v, afero.NewOsFs(), tt.f)
			if err != nil {
				t.Errorf("OpenTarGzSecrets(%v, %v, %v): %v", tt.v, afero.NewOsFs(), tt.f, err)
				continue
			}
			s = tgz
		}
		sp := secrets.SecretProducers{api.Talos: &boringSecretProducer{s}}
		vm, _ := NewSecretManager(m, sp, Filesystem(fs), MetadataFile("someta"))

		t.Run("DestroyBeforeCreated", func(t *testing.T) {
			if err := vm.Destroy(tt.v.Id); err != PathDoesNotExistError {
				t.Errorf("vm.Destroy(%v): %v", tt.v.Id, err)
			}
		})

		t.Run("Create", func(t *testing.T) {
			if err := vm.Create(tt.v); err != nil {
				t.Errorf("vm.Create(%v): %v", tt.v, err)
			}
			for {
				h, err := s.Next()
				if err == io.EOF {
					return
				}
				if err != nil {
					t.Errorf("s.Next(): %v", err)
					continue
				}
				p := path.Join(m.Path(tt.v.Id), h.Path)
				if h.FileInfo.IsDir() {
					// Assert dir
					d, err := fs.Stat(p)
					if err != nil {
						t.Errorf("fs.Stat(%v): %v", p, err)
						continue
					}
					if !d.IsDir() {
						t.Errorf("fs.Stat(%v).IsDir(): want true, got false", p)
						continue
					}
				} else {
					f, err := fs.Open(p)
					if err != nil {
						t.Errorf("fs.Open(%v): %v", p, err)
						continue
					}
					sfnv := fnv.New64a()
					dfnv := fnv.New64a()
					io.Copy(sfnv, s)
					io.Copy(dfnv, f)
					ssum := sfnv.Sum64()
					dsum := dfnv.Sum64()
					if ssum != dsum {
						t.Errorf("%v: wanted hash %v, got %v", p, fmt.Sprintf("%016x", ssum), fmt.Sprintf("%016x", dsum))
					}
				}

			}
		})

		t.Run("CreateWhenExists", func(t *testing.T) {
			if err := vm.Create(tt.v); err != PathExistsError {
				t.Errorf("vm.Create(%v): %v", tt.v, err)
			}
		})

		t.Run("List", func(t *testing.T) {
			l, err := vm.List()
			if err != nil {
				t.Errorf("vm.List(): %v", err)
			}
			if !reflect.DeepEqual(l[0], tt.v) {
				t.Errorf("vm.Get(%v): Want %v, got %v", tt.v.Id, tt.v, l[0])
			}
		})

		t.Run("Get", func(t *testing.T) {
			v, err := vm.Get(tt.v.Id)
			if err != nil {
				t.Errorf("vm.Get(%v): %v", tt.v.Id, err)
				return
			}
			if !reflect.DeepEqual(v, tt.v) {
				t.Errorf("vm.Get(%v): Want %v, got %v", tt.v.Id, tt.v, v)
			}
		})

		t.Run("Destroy", func(t *testing.T) {
			if err := vm.Destroy(tt.v.Id); err != nil {
				t.Errorf("vm.Destroy(%v): %v", tt.v.Id, err)
			}
		})
	}
}
