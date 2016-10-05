package volume

import (
	"errors"

	"github.com/negz/secret-volume/api"
)

var PathExistsError = errors.New("cannot create volume: path exists")
var PathDoesNotExistError = errors.New("cannot destroy volume: path is not a directory")
var UnknownVolumeError = errors.New("unknown volume")
var MissingMountpointError = errors.New("mountpoint does not exist")

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
