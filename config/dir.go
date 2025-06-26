package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

type Dir struct {
	Downloads string `yaml:"downloads"   json:"downloads"`
	Jobs      string `yaml:"jobs"        json:"jobs"`
}

var (
	errDirFailedToAccess = errors.New("failed to access a directory")
	errDirFailedToCreate = errors.New("failed to create a directory")
	errDirNotDirectory   = errors.New("not a directory")
)

func (cfg *Dir) Validate() error {
	errs := make([]error, 0)

	for _, dir := range []string{cfg.Downloads, cfg.Jobs} {
		if dir == "" {
			continue
		}
		if info, err := os.Stat(dir); err != nil {
			if !os.IsNotExist(err) {
				if errMkdir := os.Mkdir(dir, 0640); errMkdir != nil {
					errs = append(errs, fmt.Errorf("%w: %s: %w",
						errDirFailedToCreate, dir, err,
					))
				}
			} else {
				errs = append(errs, fmt.Errorf("%w: %s: %w",
					errDirFailedToAccess, dir, err,
				))
			}
		} else {
			if !info.IsDir() {
				errs = append(errs, fmt.Errorf("%w: %s",
					errDirNotDirectory, dir,
				))
			}
		}
	}

	return utils.FlattenErrors(errs)
}
