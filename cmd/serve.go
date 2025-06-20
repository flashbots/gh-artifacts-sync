package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/flashbots/gh-artifacts-sync/server"
)

const (
	categoryDir    = "dir"
	categoryGithub = "github"
	categoryServer = "server"
)

var (
	errGithubAppPrivateKeyCollision = fmt.Errorf(
		"cannot specify both '-%s-private-key' and '-%s-private-key-path'",
		categoryGithub, categoryGithub,
	)

	errGithubWebhookSecretCollision = fmt.Errorf(
		"cannot specify both '-%s-webhook-secret' and '-%s-webhook-secret-path'",
		categoryGithub, categoryGithub,
	)
)

func CommandServe(cfg *config.Config) *cli.Command {
	var githubAppPrivateKeyPath, githubWebhookSecretPath string

	dirFlags := []cli.Flag{ // --dir-xxx
		&cli.StringFlag{ // --dir-downloads
			Aliases:     []string{"dir.downloads"},
			Category:    strings.ToUpper(categoryDir),
			Destination: &cfg.Dir.Downloads,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryDir) + "_DOWNLOADS"},
			Name:        categoryDir + "-downloads",
			Usage:       "a `path` to the directory where downloaded artifacts will be temporarily stored",
			Value:       "./downloads",
		},

		&cli.StringFlag{ // --dir-jobs
			Aliases:     []string{"dir.jobs"},
			Category:    strings.ToUpper(categoryDir),
			Destination: &cfg.Dir.Jobs,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryDir) + "_JOBS"},
			Name:        categoryDir + "-jobs",
			Usage:       "a `path` to the directory where scheduled jobs will be persisted",
			Value:       "./jobs",
		},
	}

	githubFlags := []cli.Flag{ // --github-xxx
		altsrc.NewInt64Flag(&cli.Int64Flag{ // --github-app-id
			Aliases:     []string{"github.app.id"},
			Category:    strings.ToUpper(categoryGithub),
			Destination: &cfg.Github.App.ID,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryGithub) + "_APP_ID"},
			Name:        categoryGithub + "-app-id",
			Usage:       "github app `id`",
		}),

		altsrc.NewInt64Flag(&cli.Int64Flag{ // --github-installation-id
			Aliases:     []string{"github.app.installation_id"},
			Category:    strings.ToUpper(categoryGithub),
			Destination: &cfg.Github.App.InstallationID,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryGithub) + "_INSTALLATION_ID"},
			Name:        categoryGithub + "-installation-id",
			Usage:       "installation `id` of the github app",
		}),

		altsrc.NewStringFlag(&cli.StringFlag{ // --github-private-key
			Aliases:     []string{"github.app.private_key"},
			Category:    strings.ToUpper(categoryGithub),
			Destination: &cfg.Github.App.PrivateKey,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryGithub) + "_PRIVATE_KEY"},
			Name:        categoryGithub + "-private-key",
			Usage:       "private `key` of the github app",
		}),

		&cli.StringFlag{ // --github-private-key-path
			Category:    strings.ToUpper(categoryGithub),
			Destination: &githubAppPrivateKeyPath,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryGithub) + "_PRIVATE_KEY_PATH"},
			Name:        categoryGithub + "-private-key-path",
			Usage:       "`path` to a .pem file with private `key` of the github app",
		},

		altsrc.NewStringFlag(&cli.StringFlag{ // --github-webhook-secret
			Aliases:     []string{"github.webhook_secret"},
			Category:    strings.ToUpper(categoryGithub),
			Destination: &cfg.Github.WebhookSecret,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryGithub) + "_WEBHOOK_SECRET"},
			Name:        categoryGithub + "-webhook-secret",
			Usage:       "secret `token` for the github webhook",
		}),

		&cli.StringFlag{ // --github-webhook-secret-path
			Category:    strings.ToUpper(categoryGithub),
			Destination: &githubWebhookSecretPath,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryGithub) + "_WEBHOOK_SECRET_PATH"},
			Name:        categoryGithub + "-webhook-secret-path",
			Usage:       "`path` to a file with secret token for the github webhook",
		},
	}

	serverFlags := []cli.Flag{ // -server-xxx
		&cli.StringFlag{ // --server-listen-address
			Aliases:     []string{"server.listen_address"},
			Category:    strings.ToUpper(categoryServer),
			Destination: &cfg.Server.ListenAddress,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryServer) + "_LISTEN_ADDRESS"},
			Name:        categoryServer + "-listen-address",
			Usage:       "`host:port` for the server to listen on",
			Value:       "0.0.0.0:8080",
		},
	}

	flags := slices.Concat(
		dirFlags,
		githubFlags,
		serverFlags,
	)

	return &cli.Command{
		Name:  "serve",
		Usage: "run gh-artifacts-sync server",
		Flags: flags,

		Before: func(clictx *cli.Context) error {
			if err := altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc(flagConfig.Name))(clictx); err != nil {
				return err
			}

			if githubAppPrivateKeyPath != "" {
				if cfg.Github.App.PrivateKey != "" {
					return errGithubAppPrivateKeyCollision
				}
				bytes, err := os.ReadFile(githubAppPrivateKeyPath)
				if err != nil {
					return err
				}
				cfg.Github.App.PrivateKey = string(bytes)
			}

			if githubWebhookSecretPath != "" {
				if cfg.Github.WebhookSecret != "" {
					return errGithubWebhookSecretCollision
				}
				bytes, err := os.ReadFile(githubWebhookSecretPath)
				if err != nil {
					return err
				}
				cfg.Github.WebhookSecret = string(bytes)
			}

			return cfg.Validate()
		},

		Action: func(_ *cli.Context) error {
			s, err := server.New(cfg)
			if err != nil {
				return err
			}
			return s.Run()
		},
	}
}
