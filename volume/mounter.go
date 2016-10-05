package volume

import "github.com/negz/secret-volume/api"

// A Mounter mounts and unmounts secret volumes.
type Mounter interface {
	// Mount mounts the requested secret volume.
	Mount(*api.Volume) error
	// Unmount unmounts the secret volume specified by id.
	Unmount(id string) error
	// Path is a convenience function that returns the (theoretical) mountpoint
	// of the secret volume specified by id. Note that it does not guarantee a
	// volume with that id is currently or has ever been mounted.
	Path(id string) string
	// Root returns the parent directory of all the mounts managed by this
	// Mounter.
	Root() string
}
