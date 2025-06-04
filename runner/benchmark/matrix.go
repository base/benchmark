package benchmark

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"
)

// TestPlan represents a list of test runs to be executed.
type TestPlan struct {
	Runs         []TestRun
	Snapshot     *SnapshotDefinition
	ProofProgram *ProofProgramOptions
}

func NewTestPlanFromConfig(c TestDefinition, testFileName string) (*TestPlan, error) {
	testRuns, err := ResolveTestRunsFromMatrix(c, testFileName)
	if err != nil {
		return nil, err
	}

	// default to enabled if not set but defined
	proofProgramEnabled := c.ProofProgram != nil && (c.ProofProgram.Enabled == nil || (*c.ProofProgram.Enabled))
	var proofProgram *ProofProgramOptions
	if proofProgramEnabled {
		proofProgram = &ProofProgramOptions{
			Enabled: &proofProgramEnabled,
			Version: c.ProofProgram.Version,
			Type:    c.ProofProgram.Type,
		}
	}

	return &TestPlan{
		Runs:         testRuns,
		Snapshot:     c.Snapshot,
		ProofProgram: proofProgram,
	}, nil
}

var alphaNumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func nameToSlug(name string) string {
	return strings.ToLower(alphaNumericRegex.ReplaceAllString(name, "-"))
}

// ResolveTestRunsFromMatrix constructs a new ParamsMatrix from a config.
func ResolveTestRunsFromMatrix(c TestDefinition, testFileName string) ([]TestRun, error) {
	seenParams := make(map[string]bool)

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
		valueSelections := make(map[string]interface{})
		for j, p := range params {
			valueSelections[p.ParamType] = valuesByParam[j][currentParams[j]]
		}

		params, err := NewParamsFromValues(valueSelections)
		if err != nil {
			return nil, err
		}

		if c.Tags != nil {
			params.Tags = *c.Tags
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
