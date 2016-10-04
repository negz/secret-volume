package volume

import "github.com/negz/secret-volume/api"

type Mounter interface {
	Mount(*api.Volume) error
	Unmount(id string) error
	Path(id string) string
	Root() string
}
