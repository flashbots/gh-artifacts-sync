package main

import (
	"fmt"

	"github.com/flashbots/gh-artifacts-sync/config"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

func CommandDump(cfg *config.Config) *cli.Command {
	cmd := CommandServe(cfg)

	cmd.Name = "dump"
	cmd.Usage = "dump the effective configuration"

	cmd.Action = func(_ *cli.Context) error {
		bytes, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}

		fmt.Printf("---\n\n%s\n", string(bytes))
		return nil
	}

	return cmd
}
