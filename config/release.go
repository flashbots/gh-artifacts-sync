package config

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

type Release struct {
	regexp *regexp.Regexp `yaml:"-" json:"-"`

	AcceptDrafts      bool              `yaml:"accept_drafts"      json:"accept_drafts"`
	AcceptPrereleases bool              `yaml:"accept_prereleases" json:"accept_prereleases"`
	Assets            map[string]*Asset `yaml:"assets"             json:"assets"`
}

var (
	errReleaseInvalidAssetRegexp = errors.New("invalid asset regexp")
)

func (cfg *Release) Validate() error {
	errs := make([]error, 0)

	for regex, a := range cfg.Assets {
		if re, err := regexp.Compile(regex); err == nil {
			a.regexp = re
		} else {
			errs = append(errs, fmt.Errorf("%w: %s: %w",
				errReleaseInvalidAssetRegexp, regex, err,
			))
		}
	}

	return utils.FlattenErrors(errs)
}

func (cfg *Release) Regexp() *regexp.Regexp {
	return cfg.regexp
}
