package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

type Asset struct {
	regexp *regexp.Regexp `yaml:"-" json:"-"`

	Destinations []*Destination `yaml:"destinations" json:"destinations"`
}

var (
	errAssetInvalidDestinationType = errors.New("invalid asset destination type")
)

func (cfg *Asset) Validate() error {
	errs := make([]error, 0)

	supportedDestinationTypes := []string{
		DestinationGcpArtifactRegistryGeneric,
	}

	{ // destinations
		for _, d := range cfg.Destinations {
			if !slices.Contains(supportedDestinationTypes, d.Type) {
				errs = append(errs, fmt.Errorf("%w (must be one of: %s): %s",
					errAssetInvalidDestinationType, strings.Join(supportedDestinationTypes, ","), d.Type,
				))
			}
		}
	}

	return utils.FlattenErrors(errs)
}

func (cfg *Asset) Regexp() *regexp.Regexp {
	return cfg.regexp
}
