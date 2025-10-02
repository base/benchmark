package benchmark

import (
	"errors"

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

type BenchmarkConfig struct {
	Name                string               `yaml:"name"`
	Description         *string              `yaml:"description"`
	Benchmarks          []TestDefinition     `yaml:"benchmarks"`
	TransactionPayloads []payload.Definition `yaml:"payloads"`
}

// TestDefinition is the user-facing YAML configuration for specifying a
// matrix of benchmark runs.
type TestDefinition struct {
	InitialSnapshots []SnapshotDefinition `yaml:"initial_snapshots"`
	Metrics          *ThresholdConfig     `yaml:"metrics"`
	Tags             *map[string]string   `yaml:"tags"`
	Variables        []Param              `yaml:"variables"`
	ProofProgram     *ProofProgramOptions `yaml:"proof_program"`
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
