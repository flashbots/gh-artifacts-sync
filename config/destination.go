package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/flashbots/gh-artifacts-sync/types"
	"github.com/flashbots/gh-artifacts-sync/utils"
)

var (
	errDestinationInvalidType = errors.New("invalid destination type")
)

type Destination struct {
	Type    types.Destination `yaml:"type"    json:"type"`
	Path    string            `yaml:"path"    json:"path"`
	Package string            `yaml:"package" json:"package"`
}

func (cfg *Destination) Validate() error {
	errs := make([]error, 0)

	{ // Type
		if !slices.Contains(types.Destinations, cfg.Type) {
			errs = append(errs, fmt.Errorf("%w: %s (must be one of: %s)",
				errDestinationInvalidType,
				cfg.Type,
				strings.Join(utils.Map(types.Destinations, func(s types.Destination) string {
					return string(s)
				}), ", "),
			))
		}
	}

	return utils.FlattenErrors(errs)
}
