package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

func writeMetadataFile(t *testing.T, dir, outputDir string, runs []BenchmarkRun) {
	t.Helper()
	runDir := filepath.Join(dir, outputDir)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatal(err)
	}
	doc := BenchmarkRuns{Runs: runs}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "metadata.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func mkLocalRun(id, outputDir string, payload string, ageHours int, now time.Time) BenchmarkRun {
	created := now.Add(-time.Duration(ageHours) * time.Hour)
	return BenchmarkRun{
		ID:         id,
		SourceFile: "./mainnet-config.yml",
		OutputDir:  outputDir,
		TestName:   "Mainnet Performance Benchmark",
		TestConfig: BenchmarkTestConfig{
			BenchmarkRun:       id,
			GasLimit:           150_000_000,
			NodeType:           "builder",
			TransactionPayload: payload,
		},
		Result:    BenchmarkResult{Success: true, Complete: true},
		CreatedAt: &created,
	}
}

func TestLocalService_GetMetadata(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	l := log.New("t", "local_test")

	writeMetadataFile(t, dir, "run-a", []BenchmarkRun{mkLocalRun("a", "run-a", "transfer-only", 4, now)})
	writeMetadataFile(t, dir, "run-b", []BenchmarkRun{mkLocalRun("b", "run-b", "storage-create", 4, now)})

	svc, err := NewLocalService(dir, l)
	if err != nil {
		t.Fatal(err)
	}

	result, err := svc.GetMetadata()
	if err != nil {
		t.Fatal(err)
	}
	naturalRuns := 0
	for _, r := range result.Runs {
		if r.TestConfig.BenchmarkRun == "a" || r.TestConfig.BenchmarkRun == "b" {
			naturalRuns++
		}
	}
	if naturalRuns != 2 {
		t.Fatalf("want 2 natural runs, got %d (total=%d)", naturalRuns, len(result.Runs))
	}
}

func TestLocalService_GetMetadata_CacheHit(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	l := log.New("t", "local_test")

	writeMetadataFile(t, dir, "run-a", []BenchmarkRun{mkLocalRun("a", "run-a", "transfer-only", 4, now)})

	svc, err := NewLocalService(dir, l)
	if err != nil {
		t.Fatal(err)
	}

	r1, err := svc.GetMetadata()
	if err != nil {
		t.Fatal(err)
	}
	r2, err := svc.GetMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r2 {
		t.Error("expected the same cached pointer on second call")
	}
}

func TestLocalService_GetMetadata_InvalidatesOnNewFile(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	l := log.New("t", "local_test")

	writeMetadataFile(t, dir, "run-a", []BenchmarkRun{mkLocalRun("a", "run-a", "transfer-only", 4, now)})

	svc, err := NewLocalService(dir, l)
	if err != nil {
		t.Fatal(err)
	}

	r1, _ := svc.GetMetadata()
	countBefore := 0
	for _, r := range r1.Runs {
		if !hasSyntheticPrefix(r.TestName) {
			countBefore++
		}
	}

	writeMetadataFile(t, dir, "run-b", []BenchmarkRun{mkLocalRun("b", "run-b", "storage-create", 4, now)})

	r2, err := svc.GetMetadata()
	if err != nil {
		t.Fatal(err)
	}
	countAfter := 0
	for _, r := range r2.Runs {
		if !hasSyntheticPrefix(r.TestName) {
			countAfter++
		}
	}
	if countAfter <= countBefore {
		t.Errorf("expected more natural runs after adding a file; before=%d after=%d", countBefore, countAfter)
	}
}

func TestLocalService_GetObject(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "run-a")
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatal(err)
	}
	want := []byte(`[{"BlockNumber":1,"ExecutionMetrics":{}}]`)
	if err := os.WriteFile(filepath.Join(runDir, "metrics-sequencer.json"), want, 0644); err != nil {
		t.Fatal(err)
	}

	l := log.New("t", "local_test")
	svc, _ := NewLocalService(dir, l)

	got, err := svc.GetObject("run-a/metrics-sequencer.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLocalService_InvalidDir(t *testing.T) {
	l := log.New("t", "local_test")
	_, err := NewLocalService("/does/not/exist", l)
	if err == nil {
		t.Error("expected error for non-existent dir")
	}
}

func TestLocalService_ListAndGetLoadTests(t *testing.T) {
	dir := t.TempDir()
	network := "sepolia"
	ltDir := filepath.Join(dir, "load-tests", network)
	if err := os.MkdirAll(ltDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, ts := range []string{"2026-05-15-12-00-00", "2026-05-14-12-00-00"} {
		if err := os.WriteFile(filepath.Join(ltDir, ts+".json"), []byte(`{}`), 0644); err != nil {
			t.Fatal(err)
		}
	}

	l := log.New("t", "local_test")
	svc, _ := NewLocalService(dir, l)

	entries, err := svc.ListLoadTests(network)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if entries[0].Timestamp < entries[1].Timestamp {
		t.Error("entries should be sorted newest-first")
	}

	data, err := svc.GetLoadTest(network, entries[0].Timestamp)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{}` {
		t.Errorf("unexpected load test data: %q", data)
	}
}

func TestLocalService_PathTraversalBlocked(t *testing.T) {
	dir := t.TempDir()
	l := log.New("t", "local_test")
	svc, _ := NewLocalService(dir, l)

	for _, dangerous := range []string{
		"../etc/passwd",
		"../../secret",
		"run-a/../../outside",
	} {
		_, err := svc.GetObject(dangerous)
		if err == nil {
			t.Errorf("GetObject(%q) should have returned an error (path traversal)", dangerous)
		}
	}
}

func hasSyntheticPrefix(name string) bool {
	return len(name) > 0 && name[0] == '['
}
