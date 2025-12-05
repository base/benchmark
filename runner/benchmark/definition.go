package benchmark

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/base/base-bench/runner/payload"
)

// Param is a single dimension of a benchmark matrix. It can be a
// single value or a list of values.
type Param struct {
	Name      *string       `yaml:"name"`
	ParamType string        `yaml:"type"`
	Value     interface{}   `yaml:"value"`
	Values    []interface{} `yaml:"values"`
}

func (bp *Param) Check() error {
	if bp.Value == nil && bp.Values == nil {
		return errors.New("value or values is required")
	}
	if bp.Value != nil && bp.Values != nil {
		return errors.New("value and values cannot both be specified")
	}
	return nil
}

type ProofProgramOptions struct {
	Enabled *bool  `yaml:"enabled"`
	Version string `yaml:"version"`
	Type    string `yaml:"type"`
}

// SnapshotDefinition is the user-facing YAML configuration for specifying
// a snapshot to be restored before running a benchmark.
type SnapshotDefinition struct {
	Command           string  `yaml:"command"`
	GenesisFile       *string `yaml:"genesis_file"`
	SuperchainChainID *uint64 `yaml:"superchain_chain_id"`
	ForceClean        *bool   `yaml:"force_clean"`
}

// CreateSnapshot copies the snapshot to the output directory for the given
// node type.
func (s SnapshotDefinition) CreateSnapshot(nodeType string, outputDir string) error {
	// default to true if not set
	forceClean := s.ForceClean == nil || *s.ForceClean
	if _, err := os.Stat(outputDir); err == nil && forceClean {
		// TODO: we could reuse it here potentially
		if err := os.RemoveAll(outputDir); err != nil {
			return fmt.Errorf("failed to remove existing snapshot: %w", err)
		}
	}

	// get absolute path of outputDir
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get absolute path of outputDir: %w", err)
	}

	outputDir = path.Join(currentDir, outputDir)

	var cmdBin string
	var args []string
	// split out default args from command
	parts := strings.SplitN(s.Command, " ", 2)
	if len(parts) < 2 {
		cmdBin = parts[0]
	} else {
		cmdBin = parts[0]
		args = strings.Split(parts[1], " ")
	}

	args = append(args, nodeType, outputDir)

	cmd := exec.Command(cmdBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type BenchmarkConfig struct {
	Name                string               `yaml:"name"`
	Description         *string              `yaml:"description"`
	Benchmarks          []TestDefinition     `yaml:"benchmarks"`
	TransactionPayloads []payload.Definition `yaml:"payloads"`
}

// TestDefinition is the user-facing YAML configuration for specifying a
// matrix of benchmark runs.
type TestDefinition struct {
	Snapshot     *SnapshotDefinition  `yaml:"snapshot"`
	Metrics      *ThresholdConfig     `yaml:"metrics"`
	Tags         *map[string]string   `yaml:"tags"`
	Variables    []Param              `yaml:"variables"`
	ProofProgram *ProofProgramOptions `yaml:"proof_program"`
}

func (bc *TestDefinition) Check() error {
	for _, b := range bc.Variables {
		err := b.Check()
		if err != nil {
			return err
		}
	}
	return nil
}
