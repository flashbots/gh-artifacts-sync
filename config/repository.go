package config

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

type Repository struct {
	Containers map[string]*Container `yaml:"containers" json:"containers"`
	Releases   map[string]*Release   `yaml:"releases"   json:"releases"`
	Workflows  map[string]*Workflow  `yaml:"workflows"  json:"workflows"`
}

var (
	errRepositoryInvalidReleaseRegexp = errors.New("invalid release regexp")
)

func (cfg *Repository) Validate() error {
	errs := make([]error, 0)

	for regex, r := range cfg.Releases {
		if re, err := regexp.Compile(regex); err == nil {
			r.regexp = re
		} else {
			errs = append(errs, fmt.Errorf("%w: %s: %w",
				errRepositoryInvalidReleaseRegexp, regex, err,
			))
		}
	}

	return utils.FlattenErrors(errs)
}
