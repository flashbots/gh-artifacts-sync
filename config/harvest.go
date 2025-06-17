package config

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

type Harvest struct {
	Artifacts map[string]*Artifact `yaml:"artifacts" json:"artifacts"`
}

var (
	errHarvestInvalidRegexp = errors.New("invalid regexp")
)

func (cfg *Harvest) Validate() error {
	errs := make([]error, 0)

	for regex, a := range cfg.Artifacts {
		if re, err := regexp.Compile(regex); err == nil {
			a.regexp = re
		} else {
			errs = append(errs, fmt.Errorf("%w: %s: %w",
				errHarvestInvalidRegexp, regex, err,
			))
		}
	}

	return utils.FlattenErrors(errs)
}
