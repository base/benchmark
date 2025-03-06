package config

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
	Name      *string   `yaml:"name"`
	ParamType ParamType `yaml:"type"`
	Value     *string   `yaml:"value"`
	Values    *[]string `yaml:"values"`
}

func (bp *BenchmarkParam) Check() error {
	if bp.Value == nil && bp.Values == nil {
		return errors.New("value or values is required")
	}
	if bp.Value != nil && bp.Values != nil {
		return errors.New("value and values cannot both be specified")
	}
	return nil
}

type BenchmarkConfig struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"desciption"`
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
	for _, b := range bc.Variables {
		err := b.Check()
		if err != nil {
			return err
		}
	}
	return nil
}
