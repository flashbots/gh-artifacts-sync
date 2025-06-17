package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/flashbots/gh-artifacts-sync/utils"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Harvest map[string]map[string]*Harvest `yaml:"harvest"`

	Dir    *Dir    `yaml:"dir"    json:"dir"`
	Github *Github `yaml:"github" json:"github"`
	Log    *Log    `yaml:"log"    json:"log"`
	Server *Server `yaml:"server" json:"server"`
}

var (
	errConfigFailedToLoad = errors.New("failed to load config file")
)

func New() *Config {
	return &Config{
		Dir:    &Dir{},
		Github: &Github{App: &GithubApp{}},
		Log:    &Log{},
		Server: &Server{},
	}
}

func Load(file string) (*Config, error) {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w",
			errConfigFailedToLoad, file, err,
		)
	}
	cfg := &Config{}
	if err := yaml.UnmarshalStrict(bytes, cfg); err != nil {
		return nil, fmt.Errorf("%w: %s: %w",
			errConfigFailedToLoad, file, err,
		)
	}
	return cfg, nil
}

func (cfg *Config) SoftMerge(another *Config) {
	if another == nil {
		return
	}

	if another.Harvest != nil {
		cfg.Harvest = another.Harvest
	}
}

func (cfg *Config) Validate() error {
	errs := []error{}

	val := reflect.ValueOf(cfg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	for idx := 0; idx < val.NumField(); idx++ {
		if err := validate(val.Field(idx).Interface()); err != nil {
			errs = append(errs, err)
		}
	}

	return utils.FlattenErrors(errs)
}

func validate(item interface{}) error {
	errs := []error{}

	val := reflect.ValueOf(item)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return nil
	}

	if validatable, ok := val.Interface().(validatee); ok {
		if err := validatable.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Map:
		iter := val.MapRange()
		for iter.Next() {
			field := iter.Value()
			if field.CanInterface() {
				if err := validate(field.Interface()); err != nil {
					errs = append(errs, err)
				}
			}
		}

	case reflect.Slice, reflect.Array:
		for jdx := 0; jdx < val.Len(); jdx++ {
			field := val.Index(jdx)
			if field.CanInterface() {
				if err := validate(field.Interface()); err != nil {
					errs = append(errs, err)
				}
			}
		}

	case reflect.Struct:
		for idx := 0; idx < val.NumField(); idx++ {
			field := val.Field(idx)
			if field.CanInterface() {
				if err := validate(field.Interface()); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	return utils.FlattenErrors(errs)
}
