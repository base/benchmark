package importer

import (
	"github.com/base/base-bench/benchmark/config"
	"github.com/base/base-bench/runner/benchmark"
)

// ImportRequest represents a request to import runs
type ImportRequest struct {
	SourceMetadata  *benchmark.RunGroup
	DestMetadata    *benchmark.RunGroup
	SrcTag          *config.TagConfig
	DestTag         *config.TagConfig
	BenchmarkRunOpt BenchmarkRunOption
	NoConfirm       bool
}

// ImportResult represents the result of an import operation
type ImportResult struct {
	ImportedRuns int
	UpdatedRuns  int
	TotalRuns    int
	Success      bool
	Error        error
}

// ImportSummary provides a summary of changes to be made
type ImportSummary struct {
	ImportedRunsCount int
	ExistingRunsCount int
	SrcTagApplied     *config.TagConfig
	DestTagApplied    *config.TagConfig
	Conflicts         []string
}

// BenchmarkRunOption represents how to handle BenchmarkRun for imported runs
type BenchmarkRunOption int

const (
	BenchmarkRunAddToLast BenchmarkRunOption = iota
	BenchmarkRunCreateNew
)

// InteractivePromptState represents the state of interactive prompts
type InteractivePromptState struct {
	CurrentStep     int
	SrcTag          *config.TagConfig
	DestTag         *config.TagConfig
	BenchmarkRunOpt BenchmarkRunOption
	Confirmed       bool
	Cancelled       bool
}

const (
	StepSelectSrcTag = iota
	StepSelectDestTag
	StepSelectBenchmarkRun
	StepConfirm
	StepComplete
)
