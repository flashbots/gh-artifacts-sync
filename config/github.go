package config

import "errors"

type Github struct {
	App           *GithubApp `yaml:"app"            json:"app"`
	WebhookSecret string     `yaml:"webhook_secret" json:"webhook_secret"`
}

var (
	errGithubMustProvideWebhookSecret = errors.New("must provide github webhook secret")
)

func (cfg *Github) Validate() error {
	if cfg.WebhookSecret == "" {
		return errGithubMustProvideWebhookSecret
	}

	return nil
}
