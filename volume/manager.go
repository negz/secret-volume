package volume

import (
	"errors"

	"github.com/negz/secret-volume/api"
)

var PathExistsError = errors.New("cannot create volume: path exists")
var PathDoesNotExistError = errors.New("cannot destroy volume: path is not a directory")
var UnknownVolumeError = errors.New("unknown volume")
var MissingMountpointError = errors.New("mountpoint does not exist")

type Manager interface {
	Create(v *api.Volume) error
	Destroy(id string) error
	Get(id string) (*api.Volume, error)
	List() (api.Volumes, error)
	MetadataFile() string
}
