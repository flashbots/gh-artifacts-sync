package config

import "regexp"

type Asset struct {
	regexp *regexp.Regexp `yaml:"-" json:"-"`

	Destinations []*Destination `yaml:"destinations" json:"destinations"`
}

func (cfg *Asset) Regexp() *regexp.Regexp {
	return cfg.regexp
}
