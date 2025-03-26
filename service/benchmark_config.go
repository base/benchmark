package service

import (
	"errors"
	"fmt"
)

type BenchmarkMetric uint

const (
	BenchmarkExecutionSpeed BenchmarkMetric = iota
	BenchmarkOpProgram
)

func (b BenchmarkMetric) String() string {
	return [...]string{"execution-speed", "op-program"}[b]
}

func (b BenchmarkMetric) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b *BenchmarkMetric) UnmarshalText(text []byte) error {
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

type BenchmarkParam struct {
	ParamType ParamType `yaml:"type"`
	Value     string    `yaml:"value,omitempty"`
	Values    []string  `yaml:"values,omitempty"`
}

func (bp *BenchmarkParam) Check() error {
	if bp.Value == "" && len(bp.Values) == 0 {
		return errors.New("either value or values must be specified")
	}
	return nil
}

type BenchmarkConfig struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Benchmark   []BenchmarkMetric `yaml:"benchmark"`
	Variables   []BenchmarkParam  `yaml:"variables"`
}

func (bc *BenchmarkConfig) Check() error {
	if bc.Name == "" {
		return errors.New("name is required")
	}
	if bc.Description == "" {
		return errors.New("description is required")
	}
	if len(bc.Benchmark) == 0 {
		return errors.New("benchmark is required")
	}
	for _, v := range bc.Variables {
		if err := v.Check(); err != nil {
			return fmt.Errorf("invalid variable: %w", err)
		}
	}
	return nil
}
