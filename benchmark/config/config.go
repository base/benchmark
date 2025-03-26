package config

import (
	runnerconfig "github.com/base/base-bench/runner/config"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

// RunCmdConfig is the config needed by run command.
type RunCmdConfig struct {
	runnerconfig.Config
}

// Check validates the config.
func (c *RunCmdConfig) Check() error {
	return c.Config.Check()
}

// LogConfig returns the log config.
func (c *RunCmdConfig) LogConfig() oplog.CLIConfig {
	return c.Config.LogConfig()
}

// NewRunCmdConfig parses the RunCmdConfig from the provided flags or environment variables.
func NewRunCmdConfig(ctx *cli.Context) *RunCmdConfig {
	return &RunCmdConfig{
		Config: runnerconfig.NewConfig(ctx),
	}
}
