package main

import (
	"fmt"
	"os"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/flashbots/gh-artifacts-sync/logutils"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"go.uber.org/zap"
)

var (
	version = "development"
)

const (
	envPrefix = "GH_ARTIFACTS_SYNC_"
)

var (
	flagConfig = &cli.StringFlag{
		Name:  "config",
		Usage: "`path` to the configuration file",
	}
)

func main() {
	cfg := config.New()

	flags := []cli.Flag{
		flagConfig,

		altsrc.NewStringFlag(&cli.StringFlag{
			Aliases:     []string{"log.level"},
			Destination: &cfg.Log.Level,
			EnvVars:     []string{envPrefix + "LOG_LEVEL"},
			Name:        "log-level",
			Usage:       "logging level",
			Value:       "info",
		}),

		altsrc.NewStringFlag(&cli.StringFlag{
			Aliases:     []string{"log.mode"},
			Destination: &cfg.Log.Mode,
			EnvVars:     []string{envPrefix + "LOG_MODE"},
			Name:        "log-mode",
			Usage:       "logging mode",
			Value:       "prod",
		}),
	}

	commands := []*cli.Command{
		CommandServe(cfg),
		CommandDump(cfg),
		CommandHelp(cfg),
	}

	app := &cli.App{
		Name:    "gh-artifacts-sync",
		Usage:   "Listens to github events and synchronises artifacts, packages and releases",
		Version: version,

		Flags:          flags,
		Commands:       commands,
		DefaultCommand: commands[0].Name,

		Before: func(clictx *cli.Context) error {
			if f := clictx.String(flagConfig.Name); f != "" {
				_cfg, err := config.Load(f)
				if err != nil {
					return err
				}
				cfg.SoftMerge(_cfg)

				if err := altsrc.InitInputSourceWithContext(
					flags,
					altsrc.NewYamlSourceFromFlagFunc(flagConfig.Name),
				)(clictx); err != nil {
					return err
				}
			}

			// setup logger
			l, err := logutils.NewLogger(cfg.Log)
			if err != nil {
				return err
			}
			zap.ReplaceGlobals(l)

			return nil
		},

		Action: func(clictx *cli.Context) error {
			return cli.ShowAppHelp(clictx)
		},
	}

	defer func() {
		_ = zap.L().Sync()
	}()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "\nFailed with error:\n\n%s\n\n", err.Error())
		os.Exit(1)
	}
}
