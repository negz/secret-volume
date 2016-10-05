package volume

import (
	"path"

	"github.com/uber-go/zap"

	"github.com/negz/secret-volume/api"
)

type noopMounter struct {
	root string
}

func NewNoopMounter(root string) Mounter {
	return &noopMounter{root}
}

func (m *noopMounter) Mount(v *api.Volume) error {
	log.Debug("mount", zap.String("path", m.Path(v.ID)))
	return nil
}

func (m *noopMounter) Unmount(id string) error {
	log.Debug("unmount", zap.String("path", m.Path(id)))
	return nil
}

func (m *noopMounter) Path(id string) string {
	return path.Join(m.root, id)
}

func (m *noopMounter) Root() string {
	return m.root
}
