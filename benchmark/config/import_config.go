package config

import (
	"fmt"
	"strings"

	"github.com/base/base-bench/benchmark/flags"
	"github.com/urfave/cli/v2"
)

// ImportCmdConfig holds configuration for the import-runs command
type ImportCmdConfig struct {
	sourceFile string
	srcTag     *TagConfig
	destTag    *TagConfig
	noConfirm  bool
	outputDir  string
}

// TagConfig represents a key-value tag
type TagConfig struct {
	Key   string
	Value string
}

// NewImportCmdConfig creates a new import command configuration from CLI context
func NewImportCmdConfig(cliCtx *cli.Context) *ImportCmdConfig {
	cfg := &ImportCmdConfig{
		sourceFile: cliCtx.Args().First(), // First positional argument
		noConfirm:  cliCtx.Bool(flags.NoConfirmFlagName),
		outputDir:  cliCtx.String(flags.OutputDirFlagName),
	}

	// Parse src-tag flag
	if srcTagStr := cliCtx.String(flags.SrcTagFlagName); srcTagStr != "" {
		cfg.srcTag = parseTag(srcTagStr)
	}

	// Parse dest-tag flag
	if destTagStr := cliCtx.String(flags.DestTagFlagName); destTagStr != "" {
		cfg.destTag = parseTag(destTagStr)
	}

	return cfg
}

// parseTag parses a tag string in the format "key=value"
func parseTag(tagStr string) *TagConfig {
	parts := strings.SplitN(tagStr, "=", 2)
	if len(parts) != 2 {
		return nil
	}
	return &TagConfig{
		Key:   strings.TrimSpace(parts[0]),
		Value: strings.TrimSpace(parts[1]),
	}
}

// SourceFile returns the source metadata file path or URL
func (c *ImportCmdConfig) SourceFile() string {
	return c.sourceFile
}

// SrcTag returns the source tag configuration
func (c *ImportCmdConfig) SrcTag() *TagConfig {
	return c.srcTag
}

// DestTag returns the destination tag configuration
func (c *ImportCmdConfig) DestTag() *TagConfig {
	return c.destTag
}

// NoConfirm returns whether to skip confirmation prompts
func (c *ImportCmdConfig) NoConfirm() bool {
	return c.noConfirm
}

// OutputDir returns the output directory path
func (c *ImportCmdConfig) OutputDir() string {
	return c.outputDir
}

// Check validates the import configuration
func (c *ImportCmdConfig) Check() error {
	if c.sourceFile == "" {
		return fmt.Errorf("source file path or URL is required")
	}
	if c.outputDir == "" {
		return fmt.Errorf("output directory is required")
	}
	return nil
}
