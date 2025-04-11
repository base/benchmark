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
	DataDir() string
	OutputDir() string
	TxFuzzBinary() string
	ProxyPort() int
}

type config struct {
	logConfig     oplog.CLIConfig
	configPath    string
	dataDir       string
	outputDir     string
	clientOptions ClientOptions
	txFuzzBinary  string
	proxyPort     int
}

func NewConfig(ctx *cli.Context) Config {
	return &config{
		logConfig:     oplog.ReadCLIConfig(ctx),
		configPath:    ctx.String(appFlags.ConfigFlagName),
		dataDir:       ctx.String(appFlags.RootDirFlagName),
		outputDir:     ctx.String(appFlags.OutputDirFlagName),
		txFuzzBinary:  ctx.String(appFlags.TxFuzzBinFlagName),
		proxyPort:     ctx.Int(appFlags.ProxyPortFlagName),
		clientOptions: ReadClientOptions(ctx),
	}
}

func (c *config) ConfigPath() string {
	return c.configPath
}

func (c *config) DataDir() string {
	return c.dataDir
}

func (c *config) OutputDir() string {
	return c.outputDir
}

func (c *config) ProxyPort() int {
	return c.proxyPort
}

func (c *config) Check() error {
	if c.configPath == "" {
		return errors.New("config path is required")
	}

	// ensure file exists
	if _, err := os.Stat(c.configPath); err != nil {
		return fmt.Errorf("config file does not exist: %w", err)
	}

	if c.dataDir == "" {
		return errors.New("data dir is required")
	}

	if c.outputDir == "" {
		return errors.New("output dir is required")
	}

	return nil
}

func (c *config) LogConfig() oplog.CLIConfig {
	return c.logConfig
}

func (c *config) ClientOptions() ClientOptions {
	return c.clientOptions
}

func (c *config) TxFuzzBinary() string {
	return c.txFuzzBinary
}
