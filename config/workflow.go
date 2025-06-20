package config

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

var (
	errWorkflowInvalidArtifactRegexp = errors.New("invalid artifact regexp")
)

type Workflow struct {
	Artifacts map[string]*Artifact `yaml:"artifacts" json:"artifacts"`
}

func (cfg *Workflow) Validate() error {
	errs := make([]error, 0)

	for regex, a := range cfg.Artifacts {
		if re, err := regexp.Compile(regex); err == nil {
			a.regexp = re
		} else {
			errs = append(errs, fmt.Errorf("%w: %s: %w",
				errWorkflowInvalidArtifactRegexp, regex, err,
			))
		}
	}

	return utils.FlattenErrors(errs)
}
