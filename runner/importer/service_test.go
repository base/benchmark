package importer

import (
    "testing"
    "time"

    "github.com/base/base-bench/benchmark/config"
    "github.com/base/base-bench/runner/benchmark"
    "github.com/stretchr/testify/require"
)

// stubLogger implements the go-ethereum log.Logger interface with no-op methods for testing.
type stubLogger struct{}

func (stubLogger) Trace(string, ...interface{}) {}
func (stubLogger) Debug(string, ...interface{}) {}
func (stubLogger) Info(string, ...interface{})  {}
func (stubLogger) Warn(string, ...interface{})  {}
func (stubLogger) Error(string, ...interface{}) {}
func (stubLogger) Crit(string, ...interface{})  {}

func TestMergeMetadata_TagSemanticsAndBenchmarkRunReuse(t *testing.T) {
    now := time.Now()
    srcCreated := now.Add(2 * time.Minute)

    destMetadata := &benchmark.RunGroup{
        Runs: []benchmark.Run{
            {
                ID:   "existing",
                TestConfig: map[string]interface{}{
                    benchmark.BenchmarkRunTag: "BR-123",
                },
                CreatedAt: &now,
            },
            {
                ID: "prefilled",
                TestConfig: map[string]interface{}{
                    "instance": "keep-me",
                },
            },
        },
    }

    srcMetadata := &benchmark.RunGroup{
        CreatedAt: &srcCreated,
        Runs: []benchmark.Run{
            {ID: "imported-1"},
        },
    }

    srcTag := &config.TagConfig{Key: "instance", Value: "existing-instance"}
    destTag := &config.TagConfig{Key: "instance", Value: "imported-instance"}

    svc := &Service{config: &config.ImportCmdConfig{}, log: stubLogger{}}

    merged, summary := svc.MergeMetadata(srcMetadata, destMetadata, srcTag, destTag, BenchmarkRunAddToLast)

    require.Len(t, merged.Runs, 3)
    require.Equal(t, 1, summary.ImportedRunsCount)
    require.Equal(t, 2, summary.ExistingRunsCount)

    var imported benchmark.Run
    var existing benchmark.Run
    var prefilled benchmark.Run
    for _, run := range merged.Runs {
        switch run.ID {
        case "imported-1":
            imported = run
        case "existing":
            existing = run
        case "prefilled":
            prefilled = run
        }
    }

    // Imported runs should receive dest-tag and reuse the last BenchmarkRun ID
    require.NotNil(t, imported.TestConfig)
    require.Equal(t, destTag.Value, imported.TestConfig[destTag.Key])
    require.Equal(t, "BR-123", imported.TestConfig[benchmark.BenchmarkRunTag])
    require.NotNil(t, imported.CreatedAt)
    require.True(t, imported.CreatedAt.Equal(srcCreated))
    require.NotNil(t, imported.Result)
    require.True(t, imported.Result.Complete)

    // Existing runs should have src-tag filled only when missing
    require.Equal(t, srcTag.Value, existing.TestConfig[srcTag.Key])
    require.Equal(t, "keep-me", prefilled.TestConfig[srcTag.Key])
}

func TestMergeMetadata_CreateNewBenchmarkRunAndTags(t *testing.T) {
    srcMetadata := &benchmark.RunGroup{
        Runs: []benchmark.Run{
            {ID: "new-run"},
            {ID: "new-run-2"},
        },
    }
    destMetadata := &benchmark.RunGroup{}

    destTag := &config.TagConfig{Key: "instance", Value: "imported"}

    svc := &Service{config: &config.ImportCmdConfig{}, log: stubLogger{}}

    merged, summary := svc.MergeMetadata(srcMetadata, destMetadata, nil, destTag, BenchmarkRunCreateNew)

    require.Len(t, merged.Runs, 2)
    require.Equal(t, 2, summary.ImportedRunsCount)
    require.Equal(t, 0, summary.ExistingRunsCount)

    // All imported runs should have a BenchmarkRun ID and dest-tag applied
    var benchmarkRunID string
    for _, run := range merged.Runs {
        require.NotNil(t, run.TestConfig)
        tag, ok := run.TestConfig[benchmark.BenchmarkRunTag].(string)
        require.True(t, ok)
        require.NotEmpty(t, tag)
        if benchmarkRunID == "" {
            benchmarkRunID = tag
        } else {
            require.Equal(t, benchmarkRunID, tag)
        }
        require.Equal(t, destTag.Value, run.TestConfig[destTag.Key])
        require.NotNil(t, run.Result)
        require.True(t, run.Result.Complete)
    }
}
