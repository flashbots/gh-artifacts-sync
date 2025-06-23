package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

type Container struct {
	Destinations []*Destination `yaml:"destinations" json:"destinations"`
}

var (
	errContainerInvalidDestinationType = errors.New("invalid container destination type")
)

func (cfg *Container) Validate() error {
	errs := make([]error, 0)

	supportedDestinationTypes := []string{
		DestinationGcpArtifactRegistryDocker,
		DestinationGcpArtifactRegistryGeneric,
	}

	{ // destinations
		for _, d := range cfg.Destinations {
			if !slices.Contains(supportedDestinationTypes, d.Type) {
				errs = append(errs, fmt.Errorf("%w (must be one of: %s): %s",
					errContainerInvalidDestinationType,
					strings.Join(supportedDestinationTypes, ","),
					d.Type,
				))
			}
		}
	}

	return utils.FlattenErrors(errs)
}
