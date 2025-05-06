package benchmark

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"
)

// BenchmarkType is the type of benchmark to run, testing either sequencer speed or fault proof program speed.
type BenchmarkType uint

const (
	// BenchmarkSequencerSpeed is a type
	BenchmarkSequencerSpeed BenchmarkType = iota
	BenchmarkFaultProofProgram
)

func (b BenchmarkType) String() string {
	return [...]string{"sequencer", "fault-proof-program"}[b]
}

func (b BenchmarkType) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b *BenchmarkType) UnmarshalText(text []byte) error {
	switch string(text) {
	case "sequencer":
		*b = BenchmarkSequencerSpeed
	case "fault-proof-program":
		*b = BenchmarkFaultProofProgram
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
	ParamTypeGasLimit
	ParamTypeNumBlocks
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
	case "gas_limit":
		*b = ParamTypeGasLimit
	case "num_blocks":
		*b = ParamTypeNumBlocks
	default:
		return fmt.Errorf("invalid benchmark param type: %s", string(text))
	}
	return nil
}

// Param is a single dimension of a benchmark matrix. It can be a
// single value or a list of values.
type Param struct {
	Name      *string       `yaml:"name"`
	ParamType ParamType     `yaml:"type"`
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

// TestDefinition is the user-facing YAML configuration for specifying a
// matrix of benchmark runs.
type TestDefinition struct {
	Name        string  `yaml:"name"`
	Description string  `yaml:"desciption"`
	Variables   []Param `yaml:"variables"`
}

func (bc *TestDefinition) Check() error {
	if bc.Name == "" {
		return errors.New("name is required")
	}
	if bc.Description == "" {
		return errors.New("description is required")
	}
	for _, b := range bc.Variables {
		err := b.Check()
		if err != nil {
			return err
		}
	}
	return nil
}

// TestPlan represents a list of test runs to be executed.
type TestPlan []TestRun

func NewTestPlanFromConfig(c []TestDefinition, testFileName string) (TestPlan, error) {
	testPlan := make(TestPlan, 0, len(c))

	for _, m := range c {
		params, err := ResolveTestRunsFromMatrix(m, testFileName)
		if err != nil {
			return nil, err
		}
		testPlan = append(testPlan, params...)
	}

	return testPlan, nil
}

var alphaNumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func nameToSlug(name string) string {
	return strings.ToLower(alphaNumericRegex.ReplaceAllString(name, "-"))
}

// ResolveTestRunsFromMatrix constructs a new ParamsMatrix from a config.
func ResolveTestRunsFromMatrix(c TestDefinition, testFileName string) ([]TestRun, error) {
	seenParams := make(map[ParamType]bool)

	// Multiple payloads can run in a single benchmark.
	params := make([]Param, 0, len(c.Variables))
	for _, p := range c.Variables {
		if seenParams[p.ParamType] {
			return nil, fmt.Errorf("duplicate param type %s", p.ParamType)
		}
		seenParams[p.ParamType] = true
		params = append(params, p)
	}

	// Calculate the dimensions of the matrix for each param
	dimensions := make([]int, len(params))
	for i, p := range params {
		if p.Values != nil {
			dimensions[i] = len(p.Values)
		} else {
			dimensions[i] = 1
		}
	}

	// Create a list of values for each param
	valuesByParam := make([][]interface{}, len(params))
	for i, p := range params {
		if p.Values == nil {
			valuesByParam[i] = []interface{}{p.Value}
		} else {
			valuesByParam[i] = p.Values
		}
	}

	// Ensure total params is less than the max
	totalParams := 1
	for _, d := range dimensions {
		totalParams *= d
	}

	if totalParams > MaxTotalParams {
		return nil, fmt.Errorf("total number of params %d exceeds max %d", totalParams, MaxTotalParams)
	}

	currentParams := make([]int, len(dimensions))

	// Create the params matrix
	testParams := make([]TestRun, totalParams)

	fileNameWithoutExt := strings.TrimSuffix(path.Base(testFileName), path.Ext(testFileName))

	testOutDir := fmt.Sprintf("%s-%s-%d", nameToSlug(fileNameWithoutExt), nameToSlug(c.Name), time.Now().Unix())

	id := fmt.Sprintf("%s-%s-%d", nameToSlug(fileNameWithoutExt), nameToSlug(c.Name), time.Now().Unix())

	for i := 0; i < totalParams; i++ {
		valueSelections := make(map[ParamType]interface{})
		for j, p := range params {
			valueSelections[p.ParamType] = valuesByParam[j][currentParams[j]]
		}

		params, err := NewParamsFromValues(valueSelections)
		if err != nil {
			return nil, err
		}

		testParams[i] = TestRun{
			ID:          id,
			Params:      *params,
			OutputDir:   fmt.Sprintf("%s-%d", testOutDir, i),
			Name:        c.Name,
			Description: c.Description,
			TestFile:    testFileName,
		}

		done := true

		// Increment current params from the rightmost param
		for incIdx := len(dimensions) - 1; incIdx >= 0; incIdx-- {
			// find the next param that is incrementable
			if currentParams[incIdx] < dimensions[incIdx]-1 {
				currentParams[incIdx]++
				done = false
				break
			} else {
				// If this param is currently at the max, reset it to 0 and continue to the next param
				currentParams[incIdx] = 0
			}
		}

		if done {
			break
		}
	}

	return testParams, nil
}
