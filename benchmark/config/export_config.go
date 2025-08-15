package config

import (
	"fmt"

	"github.com/base/base-bench/benchmark/flags"
	"github.com/urfave/cli/v2"
)

// ExportCmdConfig represents the configuration for the export-to-cloud command
type ExportCmdConfig struct {
	outputDir string
	s3Bucket  string
}

// NewExportCmdConfig creates a new export command config from CLI context
func NewExportCmdConfig(cliCtx *cli.Context) *ExportCmdConfig {
	return &ExportCmdConfig{
		outputDir: cliCtx.String(flags.OutputDirFlagName),
		s3Bucket:  cliCtx.String(flags.S3BucketFlagName),
	}
}

// OutputDir returns the output directory path
func (c *ExportCmdConfig) OutputDir() string {
	return c.outputDir
}

// S3Bucket returns the S3 bucket name
func (c *ExportCmdConfig) S3Bucket() string {
	return c.s3Bucket
}

// Check validates the export command configuration
func (c *ExportCmdConfig) Check() error {
	if c.outputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if c.s3Bucket == "" {
		return fmt.Errorf("S3 bucket is required for export command")
	}

	return nil
}
