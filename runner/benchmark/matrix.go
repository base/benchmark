package benchmark

import (
	"errors"
	"fmt"
)

// Metric represents a metric that can be benchmarked. Each metric
// corresponds to a different type of benchmark.
type Metric uint

const (
	BenchmarkExecutionSpeed Metric = iota
	BenchmarkOpProgram
)

func (b Metric) String() string {
	return [...]string{"execution-speed", "op-program"}[b]
}

func (b Metric) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b *Metric) UnmarshalText(text []byte) error {
	switch string(text) {
	case "execution-speed":
		*b = BenchmarkExecutionSpeed
	case "op-program":
		*b = BenchmarkOpProgram
	default:
		return fmt.Errorf("invalid benchmark metric: %s", string(text))
	}
	return nil
}

// ParamType is an enum that specifies what variables can be specified in
// a benchmark configuration.
type ParamType uint

const (
	ParamTypeEnv ParamType = iota
	ParamTypeTxWorkload
	ParamTypeNode
)

func (b ParamType) String() string {
	return [...]string{"env", "transaction_workload", "node_type"}[b]
}

func (b ParamType) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b *ParamType) UnmarshalText(text []byte) error {
	switch string(text) {
	case "env":
		*b = ParamTypeEnv
	case "transaction_workload":
		*b = ParamTypeTxWorkload
	case "node_type":
		*b = ParamTypeNode
	default:
		return fmt.Errorf("invalid benchmark param type: %s", string(text))
	}
	return nil
}

// Param is a single dimension of a benchmark matrix. It can be a
// single value or a list of values.
type Param struct {
	Name      *string   `yaml:"name"`
	ParamType ParamType `yaml:"type"`
	Value     *string   `yaml:"value"`
	Values    *[]string `yaml:"values"`
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

// Matrix is the user-facing YAML configuration for specifying a
// matrix of benchmark runs.
type Matrix struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"desciption"`
	Benchmark   []Metric `yaml:"benchmark"`
	Variables   []Param  `yaml:"variables"`
}

func (bc *Matrix) Check() error {
	if bc.Name == "" {
		return errors.New("name is required")
	}
	if bc.Description == "" {
		return errors.New("description is required")
	}
	if len(bc.Benchmark) == 0 {
		return errors.New("benchmark is required")
	}
	for _, b := range bc.Variables {
		err := b.Check()
		if err != nil {
			return err
		}
	}
	return nil
}
