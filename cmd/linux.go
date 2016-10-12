// +build linux

package cmd

import (
	"github.com/negz/secret-volume/volume"
	"github.com/pkg/errors"

	"github.com/spf13/afero"
)

func setupFs(virt bool, root string) (volume.Mounter, afero.Fs, error) {
	if virt {
		log.Debug("Using in-memory filesystem and noop mounter")
		fs := afero.NewMemMapFs()
		if err := fs.MkdirAll(root, 0700); err != nil {
			return nil, nil, err
		}
		return volume.NewNoopMounter(root), fs, nil
	}
	tmpfs, err := volume.NewTmpFsMounter(root)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot setup tmpfs mounter")
	}

	log.Debug("Using OS filesystem and tmpfs mounter")
	return tmpfs, afero.NewOsFs(), nil
}
