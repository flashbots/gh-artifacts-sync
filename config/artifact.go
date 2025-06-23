package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

type Artifact struct {
	regexp *regexp.Regexp `yaml:"-" json:"-"`

	Destinations []*Destination `yaml:"destinations" json:"destinations"`
}

var (
	errArtifactInvalidDestinationType = errors.New("invalid artifact destination type")
)

func (cfg *Artifact) Validate() error {
	errs := make([]error, 0)

	supportedDestinationTypes := []string{
		DestinationGcpArtifactRegistryGeneric,
	}

	{ // destinations
		for _, d := range cfg.Destinations {
			if !slices.Contains(supportedDestinationTypes, d.Type) {
				errs = append(errs, fmt.Errorf("%w (must be one of: %s): %s",
					errArtifactInvalidDestinationType, strings.Join(supportedDestinationTypes, ","), d.Type,
				))
			}
		}
	}

	return utils.FlattenErrors(errs)
}

func (cfg *Artifact) Regexp() *regexp.Regexp {
	return cfg.regexp
}
