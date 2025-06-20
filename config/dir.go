package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

type Dir struct {
	Downloads string `yaml:"downloads" json:"downloads"`
	Jobs      string `yaml:"jobs"      json:"jobs"`
}

var (
	errDirNotDirectory = errors.New("not a directory")
)

func (cfg *Dir) Validate() error {
	errs := make([]error, 0)

	{ // artifacts
		if info, err := os.Stat(cfg.Downloads); err != nil {
			if !os.IsNotExist(err) {
				if errMkdir := os.Mkdir(cfg.Downloads, 0640); errMkdir != nil {
					errs = append(errs, err, errMkdir)
				}
			} else {
				errs = append(errs, err)
			}
		} else {
			if !info.IsDir() {
				errs = append(errs, fmt.Errorf("%w: %s",
					errDirNotDirectory, cfg.Downloads,
				))
			}
		}
	}

	{ // jobs
		if info, err := os.Stat(cfg.Jobs); err != nil {
			if !os.IsNotExist(err) {
				if errMkdir := os.Mkdir(cfg.Jobs, 0640); errMkdir != nil {
					errs = append(errs, err, errMkdir)
				}
			} else {
				errs = append(errs, err)
			}
		} else {
			if !info.IsDir() {
				errs = append(errs, fmt.Errorf("%w: %s",
					errDirNotDirectory, cfg.Jobs,
				))
			}
		}
	}

	return utils.FlattenErrors(errs)
}
