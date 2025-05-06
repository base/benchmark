package benchmark

import "time"

type SequencerKeyMetrics struct {
	CommonKeyMetrics
	AverageFCULatency        float64 `json:"forkChoiceUpdated"`
	AverageGetPayloadLatency float64 `json:"getPayload"`
	AverageSendTxsLatency    float64 `json:"sendTxs"`
}

type ValidatorKeyMetrics struct {
	CommonKeyMetrics
	AverageNewPayloadLatency float64 `json:"newPayload"`
}

type CommonKeyMetrics struct {
	AverageGasPerSecond float64 `json:"gasPerSecond"`
}

type BenchmarkRunResult struct {
	Success          bool                `json:"success"`
	SequencerMetrics SequencerKeyMetrics `json:"sequencerMetrics"`
	ValidatorMetrics ValidatorKeyMetrics `json:"validatorMetrics"`
}

// BenchmarkRun is the output JSON metadata for a benchmark run.
type BenchmarkRun struct {
	ID              string                 `json:"id"`
	SourceFile      string                 `json:"sourceFile"`
	OutputDir       string                 `json:"outputDir"`
	TestName        string                 `json:"testName"`
	TestDescription string                 `json:"testDescription"`
	TestConfig      map[string]interface{} `json:"testConfig"`
	Result          *BenchmarkRunResult    `json:"result"`
}

// BenchmarkRuns is the output JSON metadata file schema.
type BenchmarkRuns struct {
	Runs      []BenchmarkRun `json:"runs"`
	CreatedAt time.Time      `json:"createdAt"`
}

func (runs *BenchmarkRuns) AddResult(testIdx int, runResult BenchmarkRunResult) {
	if testIdx < 0 || testIdx >= len(runs.Runs) {
		return
	}

	runs.Runs[testIdx].Result = &runResult
}

func BenchmarkMetadataFromTestPlan(testPlan TestPlan) BenchmarkRuns {
	metadata := BenchmarkRuns{
		Runs:      make([]BenchmarkRun, 0, len(testPlan)),
		CreatedAt: time.Now(),
	}

	for _, params := range testPlan {
		metadata.Runs = append(metadata.Runs, BenchmarkRun{
			ID:              params.ID,
			SourceFile:      params.TestFile,
			TestName:        params.Name,
			TestDescription: params.Description,
			TestConfig:      params.Params.ToConfig(),
			OutputDir:       params.OutputDir,
		})
	}

	return metadata
}
