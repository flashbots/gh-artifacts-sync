package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/flashbots/gh-artifacts-sync/utils"

	cr "github.com/google/go-containerregistry/pkg/v1"
)

type Destination struct {
	Type      string   `yaml:"type"      json:"type"`
	Path      string   `yaml:"path"      json:"path"`
	Package   string   `yaml:"package"   json:"package"`
	Platforms []string `yaml:"platforms" json:"platforms"`
}

var (
	errDestinationInvalidType             = errors.New("invalid destination type")
	errDestinationDoesNotSupportPlatforms = errors.New("destination type does not support platforms option")
	errDestinationInvalidPlatform         = errors.New("invalid platform")
)

const (
	DestinationGcpArtifactRegistryDocker  = "gcp.artifactregistry.docker"
	DestinationGcpArtifactRegistryGeneric = "gcp.artifactregistry.generic"
)

func (cfg *Destination) Validate() error {
	errs := make([]error, 0)

	allDestinations := []string{
		DestinationGcpArtifactRegistryDocker,
		DestinationGcpArtifactRegistryGeneric,
	}

	{ // type
		if !slices.Contains(allDestinations, cfg.Type) {
			errs = append(errs, fmt.Errorf("%w: %s (must be one of: %s)",
				errDestinationInvalidType, cfg.Type, strings.Join(allDestinations, ","),
			))
		}
	}

	return utils.FlattenErrors(errs)
}

func (cfg *Destination) HasPlatform(p *cr.Platform) bool {
	if len(cfg.Platforms) == 0 {
		return true
	}
	return slices.Contains(cfg.Platforms, p.String())
}
