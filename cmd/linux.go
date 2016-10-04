// +build linux

package cmd

import (
	"github.com/negz/secret-volume/volume"

	"github.com/spf13/afero"
	"gopkg.in/alecthomas/kingpin.v2"
)

func setupFs(virt bool, mount string) (volume.Mounter, afero.Fs) {
	if virt {
		log.Debug("Using in-memory filesystem and noop mounter")
		return volume.NewNoopMounter(mount), afero.NewMemMapFs()
	}
	tmpfs, err := volume.NewTmpFsMounter(mount)
	kingpin.FatalIfError(err, "unable to setup tmpfs mounter")

	log.Debug("Using OS filesystem and tmpfs mounter")
	return tmpfs, afero.NewOsFs()
}
