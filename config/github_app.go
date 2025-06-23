package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/flashbots/gh-artifacts-sync/utils"
)

type GithubApp struct {
	privateKey *rsa.PrivateKey `yaml:"-" json:"-"`

	ID             int64  `yaml:"id"              json:"id"`
	InstallationID int64  `yaml:"installation_id" json:"installation_id"`
	PrivateKey     string `yaml:"private_key"     json:"private_key"`
}

var (
	errGithubAppMustProvideID             = errors.New("must provide github app id")
	errGithubAppMustProvideInstallationID = errors.New("must provide github app installation id")
	errGithubAppMustProvidePrivateKey     = errors.New("must provide github app private key")
	errGithubAppInvalidPrivateKey         = errors.New("invalid github app private key")
)

func (cfg *GithubApp) Validate() error {
	errs := []error{}

	{ // id
		if cfg.ID == 0 {
			errs = append(errs, errGithubAppMustProvideID)
		}
	}

	{ // installation_id
		if cfg.InstallationID == 0 {
			errs = append(errs, errGithubAppMustProvideInstallationID)
		}
	}

	{ // private_key
		if cfg.PrivateKey == "" {
			errs = append(errs, errGithubAppMustProvidePrivateKey)
		}

		pemBlock, _ := pem.Decode([]byte(cfg.PrivateKey))
		if pemBlock != nil && pemBlock.Bytes != nil {
			privateKey, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
			if err != nil {
				errs = append(errs, fmt.Errorf("%w: %w",
					errGithubAppInvalidPrivateKey, err,
				))
			}
			cfg.privateKey = privateKey
		} else {
			errs = append(errs, fmt.Errorf("%w: not in PEM format or empty",
				errGithubAppInvalidPrivateKey,
			))
		}
	}

	return utils.FlattenErrors(errs)
}

func (cfg *GithubApp) RsaPrivateKey() *rsa.PrivateKey {
	return cfg.privateKey
}
