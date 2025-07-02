package benchmark

import (
	"time"

	"github.com/base/base-bench/runner/network/types"
)

type RunResult struct {
	Success          bool                      `json:"success"`
	Complete         bool                      `json:"complete"`
	SequencerMetrics types.SequencerKeyMetrics `json:"sequencerMetrics"`
	ValidatorMetrics types.ValidatorKeyMetrics `json:"validatorMetrics"`
}

// Run is the output JSON metadata for a benchmark run.
type Run struct {
	ID              string                 `json:"id"`
	SourceFile      string                 `json:"sourceFile"`
	OutputDir       string                 `json:"outputDir"`
	TestName        string                 `json:"testName"`
	TestDescription string                 `json:"testDescription"`
	TestConfig      map[string]interface{} `json:"testConfig"`
	Result          *RunResult             `json:"result"`
	Thresholds      *ThresholdConfig       `json:"thresholds"`
	CreatedAt       *time.Time             `json:"createdAt"`
}

// RunGroup is a group of runs that is meant to be compared.
type RunGroup struct {
	Runs      []Run      `json:"runs"`
	CreatedAt *time.Time `json:"createdAt"` // deprecated - use Run.CreatedAt instead (only for backwards compatibility)
}

func (runs *RunGroup) AddResult(testIdx int, runResult RunResult) {
	if testIdx < 0 || testIdx >= len(runs.Runs) {
		return
	}

	runs.Runs[testIdx].Result = &runResult
}

const (
	BenchmarkRunTag = "BenchmarkRun"
)

func RunGroupFromTestPlans(testPlans []TestPlan) RunGroup {
	now := time.Now()
	metadata := RunGroup{
		Runs: make([]Run, 0),
	}

	for _, testPlan := range testPlans {
		for _, params := range testPlan.Runs {
			metadata.Runs = append(metadata.Runs, Run{
				ID:              params.ID,
				SourceFile:      params.TestFile,
				TestName:        params.Name,
				TestDescription: params.Description,
				TestConfig:      params.Params.ToConfig(),
				OutputDir:       params.OutputDir,
				Thresholds:      testPlan.Thresholds,
				CreatedAt:       &now,
			})
		}
	}

	return metadata
}
