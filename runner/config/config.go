package config

import (
	"errors"
	"fmt"
	"os"

	appFlags "github.com/base/base-bench/benchmark/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

// Config is the interface for the runtime config of the benchmark runner.
// This is everything that isn't in the config file.
type Config interface {
	Check() error
	LogConfig() oplog.CLIConfig
	ClientOptions() ClientOptions
	ConfigPath() string
	DataDir() string
	OutputDir() string
	TxFuzzBinary() string
	ProxyPort() int
	EnableS3() bool
	S3Bucket() string
	BenchmarkRunID() string
	MachineType() string
	MachineProvider() string
	MachineRegion() string
	FileSystem() string
}

type config struct {
	logConfig       oplog.CLIConfig
	configPath      string
	dataDir         string
	outputDir       string
	clientOptions   ClientOptions
	txFuzzBinary    string
	proxyPort       int
	enableS3        bool
	s3Bucket        string
	benchmarkRunID  string
	machineType     string
	machineProvider string
	machineRegion   string
	fileSystem      string
}

func NewConfig(ctx *cli.Context) Config {
	return &config{
		logConfig:       oplog.ReadCLIConfig(ctx),
		configPath:      ctx.String(appFlags.ConfigFlagName),
		dataDir:         ctx.String(appFlags.RootDirFlagName),
		outputDir:       ctx.String(appFlags.OutputDirFlagName),
		txFuzzBinary:    ctx.String(appFlags.TxFuzzBinFlagName),
		proxyPort:       ctx.Int(appFlags.ProxyPortFlagName),
		enableS3:        ctx.Bool(appFlags.EnableS3FlagName),
		s3Bucket:        ctx.String(appFlags.S3BucketFlagName),
		benchmarkRunID:  ctx.String(appFlags.BenchmarkRunIDFlagName),
		machineType:     ctx.String(appFlags.MachineTypeFlagName),
		machineProvider: ctx.String(appFlags.MachineProviderFlagName),
		machineRegion:   ctx.String(appFlags.MachineRegionFlagName),
		fileSystem:      ctx.String(appFlags.FileSystemFlagName),
		clientOptions:   ReadClientOptions(ctx),
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

func (c *config) EnableS3() bool {
	return c.enableS3
}

func (c *config) S3Bucket() string {
	return c.s3Bucket
}

func (c *config) BenchmarkRunID() string {
	return c.benchmarkRunID
}

func (c *config) MachineType() string {
	return c.machineType
}

func (c *config) MachineProvider() string {
	return c.machineProvider
}

func (c *config) MachineRegion() string {
	return c.machineRegion
}

func (c *config) FileSystem() string {
	return c.fileSystem
}
