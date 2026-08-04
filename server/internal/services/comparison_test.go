package services

import (
	"strings"
	"testing"
	"time"
)

func mkRun(id, source, payload string, gas int, version string, ageHours int, now time.Time) BenchmarkRun {
	created := now.Add(-time.Duration(ageHours) * time.Hour)
	return BenchmarkRun{
		ID:         id,
		SourceFile: source,
		OutputDir:  id + "-out",
		TestName:   "Mainnet Performance Benchmark",
		TestConfig: BenchmarkTestConfig{
			BenchmarkRun:          id,
			BlockTimeMilliseconds: 1000,
			GasLimit:              gas,
			NodeType:              "builder",
			TransactionPayload:    payload,
			ClientVersion:         version,
		},
		Result:    BenchmarkResult{Success: true, Complete: true, ClientVersion: version},
		CreatedAt: &created,
	}
}

func TestSynthesize_VersionGroupForMainnet(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	runs := []BenchmarkRun{
		mkRun("a", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1.0.0", 4, now),
		mkRun("b", "./mainnet-config.yml", "transfer-only", 150_000_000, "v2.0.0", 4, now),
	}
	synth := synthesizeComparisonGroups(runs, now)

	versionGroup := filterByBenchmarkRunPrefix(synth, "compare-version-mainnet")
	if len(versionGroup) != 2 {
		t.Fatalf("want 2 synthetic version-group runs, got %d", len(versionGroup))
	}
	for _, r := range versionGroup {
		if !strings.HasPrefix(r.TestName, "[Compare: Versions]") {
			t.Errorf("synthetic run TestName %q missing [Compare: Versions] prefix", r.TestName)
		}
	}
}

func TestSynthesize_TimeGroupForMainnet(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	runs := []BenchmarkRun{
		mkRun("hot", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1", 4, now),
		mkRun("warm", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1", 72, now),
	}
	synth := synthesizeComparisonGroups(runs, now)

	timeGroup := filterByBenchmarkRunPrefix(synth, "compare-time-mainnet")
	if len(timeGroup) != 2 {
		t.Fatalf("want 2 synthetic time-group runs (one per bucket), got %d", len(timeGroup))
	}
	for _, r := range timeGroup {
		if !strings.HasPrefix(r.TestName, "[Compare: Time]") {
			t.Errorf("synthetic run TestName %q missing [Compare: Time] prefix", r.TestName)
		}
	}
}

func TestSynthesize_DropsSingleBucketComparison(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	// All runs at the same version → version comparison should be skipped.
	runs := []BenchmarkRun{
		mkRun("a", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1.0.0", 4, now),
		mkRun("b", "./mainnet-config.yml", "storage-create", 150_000_000, "v1.0.0", 4, now),
	}
	synth := synthesizeComparisonGroups(runs, now)
	if got := len(filterByBenchmarkRunPrefix(synth, "compare-version-mainnet")); got != 0 {
		t.Errorf("single-version cohort should produce no version-comparison runs, got %d", got)
	}
}

func TestSynthesize_DoesNotMutateSource(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	source := []BenchmarkRun{
		mkRun("a", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1.0.0", 4, now),
		mkRun("b", "./mainnet-config.yml", "transfer-only", 150_000_000, "v2.0.0", 4, now),
	}
	originalIDs := []string{source[0].TestConfig.BenchmarkRun, source[1].TestConfig.BenchmarkRun}
	originalNames := []string{source[0].TestName, source[1].TestName}

	_ = synthesizeComparisonGroups(source, now)

	for i, r := range source {
		if r.TestConfig.BenchmarkRun != originalIDs[i] {
			t.Errorf("source run %d BenchmarkRun mutated: %q != %q", i, r.TestConfig.BenchmarkRun, originalIDs[i])
		}
		if r.TestName != originalNames[i] {
			t.Errorf("source run %d TestName mutated: %q != %q", i, r.TestName, originalNames[i])
		}
	}
}

func TestSynthesize_VersionGroupSkipsRunsWithoutVersion(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	runs := []BenchmarkRun{
		mkRun("a", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1.0.0", 4, now),
		mkRun("b", "./mainnet-config.yml", "transfer-only", 150_000_000, "v2.0.0", 4, now),
		mkRun("c", "./mainnet-config.yml", "transfer-only", 150_000_000, "", 4, now),
	}
	runs[2].Result.ClientVersion = ""
	synth := synthesizeComparisonGroups(runs, now)
	versionGroup := filterByBenchmarkRunPrefix(synth, "compare-version-mainnet")
	if len(versionGroup) != 2 {
		t.Fatalf("want 2 runs (one per non-empty version), got %d", len(versionGroup))
	}
}

func TestSynthesize_LatestPerVariantPerVersion(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	runs := []BenchmarkRun{
		// Two runs of v1.0.0 same variant — synthetic group should keep only the newer.
		mkRun("v1-old", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1.0.0", 48, now),
		mkRun("v1-new", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1.0.0", 6, now),
		mkRun("v2", "./mainnet-config.yml", "transfer-only", 150_000_000, "v2.0.0", 24, now),
	}
	synth := synthesizeComparisonGroups(runs, now)
	versionGroup := filterByBenchmarkRunPrefix(synth, "compare-version-mainnet")
	if len(versionGroup) != 2 {
		t.Fatalf("want 2 runs (latest per version), got %d", len(versionGroup))
	}
	gotIDs := map[string]bool{}
	for _, r := range versionGroup {
		gotIDs[r.ID] = true
	}
	if gotIDs["v1-old"] {
		t.Errorf("v1-old should have been replaced by v1-new")
	}
	if !gotIDs["v1-new"] || !gotIDs["v2"] {
		t.Errorf("expected v1-new and v2 in synthetic group, got %v", gotIDs)
	}
}

func TestSynthesize_NetworkScoping(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	runs := []BenchmarkRun{
		mkRun("a", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1", 4, now),
		mkRun("b", "./mainnet-config.yml", "transfer-only", 150_000_000, "v2", 4, now),
		// devnet runs must not leak into the mainnet synthetic group.
		mkRun("c", "./devnet-config.yml", "transfer-only", 150_000_000, "v1", 4, now),
		mkRun("d", "./devnet-config.yml", "transfer-only", 150_000_000, "v2", 4, now),
	}
	synth := synthesizeComparisonGroups(runs, now)

	mainnet := filterByBenchmarkRunPrefix(synth, "compare-version-mainnet")
	devnet := filterByBenchmarkRunPrefix(synth, "compare-version-devnet")

	for _, r := range mainnet {
		if !strings.Contains(r.SourceFile, "mainnet") {
			t.Errorf("mainnet group leaked a non-mainnet run: %s", r.SourceFile)
		}
	}
	for _, r := range devnet {
		if !strings.Contains(r.SourceFile, "devnet") {
			t.Errorf("devnet group leaked a non-devnet run: %s", r.SourceFile)
		}
	}
	if len(mainnet) == 0 || len(devnet) == 0 {
		t.Fatalf("both networks should have produced groups; got mainnet=%d devnet=%d", len(mainnet), len(devnet))
	}
}

func TestSynthesize_StableID(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	runs := []BenchmarkRun{
		mkRun("a", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1", 4, now),
		mkRun("b", "./mainnet-config.yml", "transfer-only", 150_000_000, "v2", 4, now),
	}
	first := synthesizeComparisonGroups(runs, now)
	second := synthesizeComparisonGroups(runs, now.Add(7*24*time.Hour))

	id1 := first[0].TestConfig.BenchmarkRun
	id2 := second[0].TestConfig.BenchmarkRun
	if id1 != id2 {
		t.Errorf("synthetic IDs must be stable across calls; got %q vs %q", id1, id2)
	}
}

// TestSynthesize_TimeGroupStampsBucketAndDate covers the bug where
// time-comparison runs were indistinguishable to the frontend's
// chart split-by. Every run in a time-comparison group must carry
// distinct values in at least one testConfig key, otherwise the
// frontend has nothing to plot as separate series.
func TestSynthesize_TimeGroupStampsBucketAndDate(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	runs := []BenchmarkRun{
		mkRun("hot", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1", 4, now),
		mkRun("warm", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1", 72, now),
		mkRun("cold", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1", 14*24, now),
	}
	synth := synthesizeComparisonGroups(runs, now)
	timeGroup := filterByBenchmarkRunPrefix(synth, "compare-time-mainnet")
	if len(timeGroup) != 3 {
		t.Fatalf("want 3 runs across 1d/1w/1m buckets, got %d", len(timeGroup))
	}

	buckets := map[string]bool{}
	for _, r := range timeGroup {
		if r.TestConfig.TimeBucket == "" {
			t.Errorf("synthetic time-group run is missing TimeBucket: %+v", r)
		}
		buckets[r.TestConfig.TimeBucket] = true
	}
	if len(buckets) != 3 {
		t.Errorf("expected 3 distinct TimeBucket values (1d/1w/1m), got %d: %v", len(buckets), buckets)
	}
	for _, want := range []string{"1d", "1w", "1m"} {
		if !buckets[want] {
			t.Errorf("missing TimeBucket=%q", want)
		}
	}
}

// TestSynthesize_VersionGroupOmitsTimeBucket confirms that
// version-comparison runs do NOT get TimeBucket stamped (it would be
// misleading — the split axis is ClientVersion, not a time window).
func TestSynthesize_VersionGroupOmitsTimeBucket(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	runs := []BenchmarkRun{
		mkRun("a", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1.0.0", 4, now),
		mkRun("b", "./mainnet-config.yml", "transfer-only", 150_000_000, "v2.0.0", 4, now),
	}
	synth := synthesizeComparisonGroups(runs, now)
	versionGroup := filterByBenchmarkRunPrefix(synth, "compare-version-mainnet")
	if len(versionGroup) != 2 {
		t.Fatalf("want 2 runs (one per version), got %d", len(versionGroup))
	}
	for _, r := range versionGroup {
		if r.TestConfig.TimeBucket != "" {
			t.Errorf("version-group run should NOT have TimeBucket; got %q on run %s", r.TestConfig.TimeBucket, r.ID)
		}
	}
}

// TestSynthesize_SourceRunsLackStampedFields verifies the omitempty
// boundary: TimeBucket must never appear on the source runs (the
// natural per-run pages), only on synthetic clones. A leak here
// would surface phantom dropdown values on natural-run pages.
func TestSynthesize_SourceRunsLackStampedFields(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	source := []BenchmarkRun{
		mkRun("a", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1", 4, now),
		mkRun("b", "./mainnet-config.yml", "transfer-only", 150_000_000, "v2", 4, now),
	}
	_ = synthesizeComparisonGroups(source, now)

	for _, r := range source {
		if r.TestConfig.TimeBucket != "" {
			t.Errorf("source run %s has unexpected TimeBucket=%q after synthesis", r.ID, r.TestConfig.TimeBucket)
		}
	}
}

// TestSynthesize_ClonesShareNowCreatedAt covers the dropdown-label
// fix: synthetic-group clones must all carry the same `now`-time
// createdAt so the frontend's dropdown entry shows one coherent
// timestamp (the comparison view's freshness) instead of an
// arbitrary source run's timestamp.
func TestSynthesize_ClonesShareNowCreatedAt(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	runs := []BenchmarkRun{
		mkRun("hot", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1", 4, now),
		mkRun("warm", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1", 72, now),
	}
	synth := synthesizeComparisonGroups(runs, now)
	timeGroup := filterByBenchmarkRunPrefix(synth, "compare-time-mainnet")
	if len(timeGroup) != 2 {
		t.Fatalf("want 2 time-group runs, got %d", len(timeGroup))
	}
	for _, r := range timeGroup {
		if r.CreatedAt == nil {
			t.Fatalf("synthetic run missing CreatedAt: %+v", r)
		}
		if !r.CreatedAt.Equal(now) {
			t.Errorf("synthetic run CreatedAt should be now=%v, got %v (run %s)", now, *r.CreatedAt, r.ID)
		}
	}
	// Source runs must still carry their original CreatedAts.
	if runs[0].CreatedAt.Equal(now) {
		t.Error("source run CreatedAt was overwritten — clones must not leak back into sources")
	}
}

func TestSynthesize_MonthlyPrefixDoesNotIsolateRuns(t *testing.T) {
	// Regression: applyRetentionPolicy adds a "[Monthly - <Mon YYYY>] "
	// prefix to runs preserved in the older retention bucket. If the
	// synthesizer groups by raw TestName, those prefixed runs end up
	// in a 1-variant cohort and never participate in any comparison.
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	monthly := mkRun("monthly", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1.0.0", 14*24, now)
	monthly.TestName = "[Monthly - May 2026] Mainnet Performance Benchmark"
	recent := mkRun("recent", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1.0.0", 4, now)
	recent.TestName = "Mainnet Performance Benchmark"
	weekRun := mkRun("week", "./mainnet-config.yml", "transfer-only", 150_000_000, "v1.0.0", 72, now)
	weekRun.TestName = "Mainnet Performance Benchmark"

	synth := synthesizeComparisonGroups([]BenchmarkRun{monthly, recent, weekRun}, now)

	timeGroup := filterByBenchmarkRunPrefix(synth, "compare-time-mainnet")
	if len(timeGroup) != 3 {
		t.Fatalf("want 3 runs across 1d/1w/1m buckets after canonicalization, got %d", len(timeGroup))
	}
	gotIDs := map[string]bool{}
	for _, r := range timeGroup {
		gotIDs[r.ID] = true
	}
	for _, want := range []string{"monthly", "recent", "week"} {
		if !gotIDs[want] {
			t.Errorf("synthetic time group should include source run %q (was the [Monthly] prefix isolating it?)", want)
		}
	}
}

func TestCanonicalTestName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Base Mainnet Performance", "Base Mainnet Performance"},
		{"[Monthly - May 2026] Base Mainnet Performance", "Base Mainnet Performance"},
		{"[Monthly - Mar 2026] Anything Goes Here", "Anything Goes Here"},
		{"[Monthly - malformed", "[Monthly - malformed"},
		{"", ""},
	}
	for _, c := range cases {
		if got := canonicalTestName(c.in); got != c.want {
			t.Errorf("canonicalTestName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"mainnet-Base Mainnet Performance", "mainnet-base-mainnet-performance"},
		{"  ___weird  chars!!!", "weird-chars"},
		{"---", ""},
		{"already-slug", "already-slug"},
	}
	for _, c := range cases {
		if got := slugify(c.in); got != c.want {
			t.Errorf("slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// filterByBenchmarkRunPrefix is a test helper: returns the runs whose
// synthetic BenchmarkRun ID starts with the given prefix.
func filterByBenchmarkRunPrefix(runs []BenchmarkRun, prefix string) []BenchmarkRun {
	var out []BenchmarkRun
	for _, r := range runs {
		if strings.HasPrefix(r.TestConfig.BenchmarkRun, prefix) {
			out = append(out, r)
		}
	}
	return out
}
