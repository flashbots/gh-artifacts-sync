package config

import (
	"errors"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

type GithubApp struct {
	ID             int64  `yaml:"id"              json:"id"`
	InstallationID int64  `yaml:"installation_id" json:"installation_id"`
	PrivateKey     string `yaml:"private_key"     json:"private_key"`
}

var (
	errGithubAppMustProvideID             = errors.New("must provide github app id")
	errGithubAppMustProvideInstallationID = errors.New("must provide github app installation id")
	errGithubAppMustProvidePrivateKey     = errors.New("must provide github app private key")
)

func (cfg *GithubApp) Validate() error {
	errs := []error{}

	if cfg.ID == 0 {
		errs = append(errs, errGithubAppMustProvideID)
	}

	if cfg.InstallationID == 0 {
		errs = append(errs, errGithubAppMustProvideInstallationID)
	}

	if cfg.PrivateKey == "" {
		errs = append(errs, errGithubAppMustProvidePrivateKey)
	}

	return utils.FlattenErrors(errs)
}
