package config

import (
	"errors"
	"fmt"
	"os"

	appFlags "github.com/base/base-bench/benchmark/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

// Config is the interface for the config of the benchmark runner.
type Config interface {
	Check() error
	LogConfig() oplog.CLIConfig
	ClientOptions() ClientOptions
	ConfigPath() string
	RootDir() string
}

type config struct {
	logConfig     oplog.CLIConfig
	configPath    string
	rootDir       string
	clientOptions ClientOptions
}

func NewConfig(ctx *cli.Context) Config {
	return &config{
		logConfig:     oplog.ReadCLIConfig(ctx),
		configPath:    ctx.String(appFlags.ConfigFlagName),
		rootDir:       ctx.String(appFlags.RootDirFlagName),
		clientOptions: ReadClientOptions(ctx),
	}
}

func (c *config) ConfigPath() string {
	return c.configPath
}

func (c *config) RootDir() string {
	return c.rootDir
}

func (c *config) Check() error {
	if c.configPath == "" {
		return errors.New("config path is required")
	}

	// ensure file exists
	if _, err := os.Stat(c.configPath); err != nil {
		return fmt.Errorf("config file does not exist: %w", err)
	}
	return nil
}

func (c *config) LogConfig() oplog.CLIConfig {
	return c.logConfig
}

func (c *config) ClientOptions() ClientOptions {
	return c.clientOptions
}
