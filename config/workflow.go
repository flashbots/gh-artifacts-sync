package config

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

type Workflow struct {
	Actors    []string             `yaml:"actors"    json:"actors"`
	Artifacts map[string]*Artifact `yaml:"artifacts" json:"artifacts"`

	actors map[string]struct{} `yaml:"-" json:"-"`
}

var (
	errWorkflowInvalidArtifactRegexp = errors.New("invalid artifact regexp")
)

func (cfg *Workflow) Validate() error {
	errs := make([]error, 0)

	{ // actors
		cfg.actors = make(map[string]struct{}, len(cfg.Actors))
		for _, a := range cfg.Actors {
			cfg.actors[a] = struct{}{}
		}
	}

	{ // artifacts
		for regex, a := range cfg.Artifacts {
			if re, err := regexp.Compile(regex); err == nil {
				a.regexp = re
			} else {
				errs = append(errs, fmt.Errorf("%w: %s: %w",
					errWorkflowInvalidArtifactRegexp, regex, err,
				))
			}
		}
	}

	return utils.FlattenErrors(errs)
}

func (cfg *Workflow) HasActor(a string) bool {
	_, has := cfg.actors[a]
	return has
}
