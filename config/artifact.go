package config

import "regexp"

type Artifact struct {
	regexp *regexp.Regexp `yaml:"-" json:"-"`

	Destinations []*Destination `yaml:"destinations" json:"destinations"`
}

func (cfg *Artifact) Regexp() *regexp.Regexp {
	return cfg.regexp
}
