// +build linux

package volume

import (
	"fmt"
	"path"

	"github.com/negz/secret-volume/api"

	"github.com/uber-go/zap"
	"golang.org/x/sys/unix"
)

type tmpFsMounter struct {
	root   string
	max    uint
	mode   uint32
	mflags uintptr
	uflags int
}

type TmpFsMounterOption func(*tmpFsMounter) error

func MountpointMode(md uint32) TmpFsMounterOption {
	return func(m *tmpFsMounter) error {
		m.mode = md
		return nil
	}
}

func MaxSizeMb(mb uint) TmpFsMounterOption {
	return func(m *tmpFsMounter) error {
		m.max = mb
		return nil
	}
}

func MountFlags(flags uintptr) TmpFsMounterOption {
	return func(m *tmpFsMounter) error {
		m.mflags = flags
		return nil
	}
}

func UnmountFlags(flags int) TmpFsMounterOption {
	return func(m *tmpFsMounter) error {
		m.uflags = flags
		return nil
	}
}

func NewTmpFsMounter(root string, mo ...TmpFsMounterOption) (Mounter, error) {
	m := &tmpFsMounter{root, 100, 700, unix.MS_NOSUID | unix.MS_NODEV | unix.MS_NOEXEC, 0}
	for _, o := range mo {
		if err := o(m); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (m *tmpFsMounter) Path(id string) string {
	return path.Join(m.root, id)
}

func (m *tmpFsMounter) Root() string {
	return m.root
}

func (m *tmpFsMounter) flags() string {
	return fmt.Sprintf("size=%vM,mode=%v", m.max, int(m.mode))
}

func (m *tmpFsMounter) Mount(v *api.Volume) error {
	f := m.flags()
	log.Debug("mount", zap.String("path", m.Path(v.ID)), zap.String("flags", f))
	return unix.Mount("tmpfs", m.Path(v.ID), "tmpfs", m.mflags, f)
}

func (m *tmpFsMounter) Unmount(id string) error {
	log.Debug("unmount", zap.String("path", m.Path(id)))
	return unix.Unmount(m.Path(id), m.uflags)
}
