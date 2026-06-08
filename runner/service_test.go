package runner

import (
	"testing"

	"github.com/base/base-bench/runner/benchmark"
)

func TestApplyClientVersion_AutoDetected(t *testing.T) {
	run := &benchmark.Run{}
	result := &benchmark.RunResult{ClientVersion: "reth/1.7.0"}
	applyClientVersion(run, result, "")

	if got := result.ClientVersion; got != "reth/1.7.0" {
		t.Fatalf("result.ClientVersion=%q want %q", got, "reth/1.7.0")
	}
	if got := run.TestConfig["ClientVersion"]; got != "reth/1.7.0" {
		t.Fatalf("TestConfig[ClientVersion]=%v want %q", got, "reth/1.7.0")
	}
}

func TestApplyClientVersion_OverrideWins(t *testing.T) {
	run := &benchmark.Run{}
	result := &benchmark.RunResult{ClientVersion: "auto-detected"}
	applyClientVersion(run, result, "v1.2.3-rc1")

	if got := result.ClientVersion; got != "v1.2.3-rc1" {
		t.Fatalf("override should win on result, got %q", got)
	}
	if got := run.TestConfig["ClientVersion"]; got != "v1.2.3-rc1" {
		t.Fatalf("override should win in TestConfig, got %v", got)
	}
}

func TestApplyClientVersion_OverrideFillsEmptyAuto(t *testing.T) {
	run := &benchmark.Run{}
	result := &benchmark.RunResult{}
	applyClientVersion(run, result, "from-env")

	if got := run.TestConfig["ClientVersion"]; got != "from-env" {
		t.Fatalf("override should populate when auto-detect was empty, got %v", got)
	}
}

func TestApplyClientVersion_NeitherSet(t *testing.T) {
	run := &benchmark.Run{}
	result := &benchmark.RunResult{}
	applyClientVersion(run, result, "")

	if run.TestConfig != nil {
		t.Fatalf("TestConfig must not be allocated when no version is available, got %+v", run.TestConfig)
	}
	if result.ClientVersion != "" {
		t.Fatalf("result.ClientVersion must stay empty, got %q", result.ClientVersion)
	}
}

func TestApplyClientVersion_PreservesOtherTestConfigKeys(t *testing.T) {
	run := &benchmark.Run{TestConfig: map[string]interface{}{"BenchmarkRun": "test-123", "GasLimit": 150_000_000}}
	result := &benchmark.RunResult{ClientVersion: "reth/1.7.0"}
	applyClientVersion(run, result, "")

	if got := run.TestConfig["BenchmarkRun"]; got != "test-123" {
		t.Errorf("pre-existing BenchmarkRun key should survive, got %v", got)
	}
	if got := run.TestConfig["GasLimit"]; got != 150_000_000 {
		t.Errorf("pre-existing GasLimit key should survive, got %v", got)
	}
	if got := run.TestConfig["ClientVersion"]; got != "reth/1.7.0" {
		t.Errorf("ClientVersion not stamped, got %v", got)
	}
}
