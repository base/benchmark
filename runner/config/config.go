package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/base/base-bench/benchmark/flags"
	"github.com/base/base-bench/clients/types"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

type Config interface {
	Check() error
	LogConfig() oplog.CLIConfig
	ClientOptions() types.ClientOptions
	ConfigPath() string
	RootDir() string
}

type config struct {
	logConfig     oplog.CLIConfig
	configPath    string
	rootDir       string
	clientOptions types.ClientOptions
}

func NewConfig(ctx *cli.Context) Config {
	return &config{
		logConfig:     oplog.ReadCLIConfig(ctx),
		configPath:    ctx.String(flags.ConfigFlagName),
		rootDir:       ctx.String(flags.RootDirFlagName),
		clientOptions: types.ReadClientOptions(ctx),
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

func (c *config) ClientOptions() types.ClientOptions {
	return c.clientOptions
}
