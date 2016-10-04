// +build !linux

package cmd

import (
	"github.com/negz/secret-volume/volume"

	"github.com/spf13/afero"
)

func setupFs(_ bool, mount string) (volume.Mounter, afero.Fs) {
	// The tmpfs mounter will only build on Linux
	log.Debug("Forcing in-memory filesystem and noop mounter due to non-Linux environment")
	return volume.NewNoopMounter(mount), afero.NewMemMapFs()
}
