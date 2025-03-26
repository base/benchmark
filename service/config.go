package service

import (
	"errors"
	"fmt"
	"os"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

type Config interface {
	Check() error
	LogConfig() oplog.CLIConfig
	ConfigPath() string
	Benchmarks() []BenchmarkConfig
}

type config struct {
	logConfig  oplog.CLIConfig
	configPath string
	benchmarks []BenchmarkConfig
}

func NewConfig(ctx *cli.Context) Config {
	return &config{
		logConfig:  oplog.ReadCLIConfig(ctx),
		configPath: ctx.String("config"),
	}
}

func (c *config) ConfigPath() string {
	return c.configPath
}

func (c *config) LogConfig() oplog.CLIConfig {
	return c.logConfig
}

func (c *config) Benchmarks() []BenchmarkConfig {
	return c.benchmarks
}

func (c *config) Check() error {
	if c.configPath == "" {
		return errors.New("config path is required")
	}

	// ensure file exists
	if _, err := os.Stat(c.configPath); err != nil {
		return fmt.Errorf("config file does not exist: %w", err)
	}

	// Load and validate benchmarks
	if err := c.loadBenchmarks(); err != nil {
		return fmt.Errorf("failed to load benchmarks: %w", err)
	}

	return nil
}

func (c *config) loadBenchmarks() error {
	file, err := os.OpenFile(c.configPath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var configs []BenchmarkConfig
	if err := yaml.NewDecoder(file).Decode(&configs); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	// Validate each benchmark
	for i, bc := range configs {
		if err := bc.Check(); err != nil {
			return fmt.Errorf("invalid benchmark at index %d: %w", i, err)
		}
	}

	c.benchmarks = configs
	return nil
}
