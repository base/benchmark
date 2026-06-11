package services

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Comparison groups expose multi-version and multi-time-window views
// of an existing benchmark series as if they were ordinary
// `BenchmarkRun` IDs. The frontend consumes the same `metadata.json`
// it always has — selecting a synthetic ID from the dropdown loads
// runs sourced from across versions or time windows. To draw them as
// separate chart series, the user picks "Show Line Per: ClientVersion"
// (for version comparisons) in the existing chart UI; no frontend
// change required.
//
// Synthetic runs are deep clones of source runs with rewritten
// TestConfig["BenchmarkRun"] and prefixed TestName. The source runs
// are never mutated and continue to appear under their natural IDs.

// comparisonKind picks which dimension a synthetic group splits on.
type comparisonKind int

const (
	compareByTime comparisonKind = iota
	compareByVersion
)

// comparisonGroupConfig defines one cohort to manufacture synthetic
// groups for. Cohorts are scoped by the substring matched against
// `sourceFile` (network) — every distinct testName within that cohort
// gets its own synthetic groups so two different mainnet test series
// don't get conflated.
//
// Hardcoded for v1; a future iteration may load this from
// s3://.../comparisons/groups.json so operators can change the set
// without redeploying the API.
type comparisonGroupConfig struct {
	// networkSubstr is matched case-insensitively against
	// run.SourceFile. A run participates in this cohort iff its
	// SourceFile contains the substring. "" means "any network".
	networkSubstr string
	// kinds is the comparison dimensions to manufacture for this
	// cohort. Each kind that produces fewer than 2 distinct buckets
	// is dropped (a comparison of one thing isn't a comparison).
	kinds []comparisonKind
}

// hardcodedComparisonGroups defines the v1 set: each canonical
// network gets both a by-version and a by-time comparison. The
// generator silently skips any (cohort, kind) pair that doesn't have
// enough data to be a meaningful comparison, so listing all three
// networks here is harmless even when only one is populated.
var hardcodedComparisonGroups = []comparisonGroupConfig{
	{networkSubstr: "mainnet", kinds: []comparisonKind{compareByVersion, compareByTime}},
	{networkSubstr: "sepolia", kinds: []comparisonKind{compareByVersion, compareByTime}},
	{networkSubstr: "testnet", kinds: []comparisonKind{compareByVersion, compareByTime}},
	{networkSubstr: "devnet", kinds: []comparisonKind{compareByVersion, compareByTime}},
}

// timeBuckets defines the rolling windows for compareByTime. Each is
// a half-open interval [Newer, Older) measured as "ago" relative to
// `now`. Ordered newest-first because the user typically wants to
// see "today" alongside "last week" and "last month".
var timeBuckets = []struct {
	Label string
	Newer time.Duration
	Older time.Duration
}{
	{"1d", 0, 24 * time.Hour},
	{"1w", 24 * time.Hour, 7 * 24 * time.Hour},
	{"1m", 7 * 24 * time.Hour, 30 * 24 * time.Hour},
}

// synthesizeComparisonGroups returns synthetic runs to append to the
// merged metadata. It never mutates the input. Source runs are deep
// cloned (TestConfig map is rebuilt; pointer fields are shared since
// they're treated as immutable everywhere downstream).
func synthesizeComparisonGroups(runs []BenchmarkRun, now time.Time) []BenchmarkRun {
	var synthetic []BenchmarkRun
	for _, cfg := range hardcodedComparisonGroups {
		cohort := filterByNetwork(runs, cfg.networkSubstr)
		if len(cohort) == 0 {
			continue
		}
		// Group cohort runs by canonical testName (with the
		// applyRetentionPolicy "[Monthly - Mon YYYY] " prefix
		// stripped) so the monthly survivor of a series lands in
		// the same cohort as its 1d/1w siblings. Otherwise the
		// monthly run gets isolated into a 1-variant cohort and
		// every comparison degenerates.
		byTestName := map[string][]BenchmarkRun{}
		for _, r := range cohort {
			byTestName[canonicalTestName(r.TestName)] = append(byTestName[canonicalTestName(r.TestName)], r)
		}
		for testName, sameNameRuns := range byTestName {
			for _, kind := range cfg.kinds {
				synthetic = append(synthetic, manufactureGroup(sameNameRuns, cfg.networkSubstr, testName, kind, now)...)
			}
		}
	}
	return synthetic
}

// canonicalTestName strips the "[Monthly - <Mon YYYY>] " prefix that
// applyRetentionPolicy may have added so two runs of the same series
// land in the same cohort regardless of which retention bucket they
// were preserved under. The format is fixed by s3.go::applyRetentionPolicy
// and reproduced here intentionally — when that format changes, both
// places must update together.
func canonicalTestName(name string) string {
	if !strings.HasPrefix(name, "[Monthly - ") {
		return name
	}
	if end := strings.Index(name, "] "); end >= 0 {
		return name[end+2:]
	}
	return name
}

// pickedRun pairs a chosen source run with the bucket label it was
// chosen for. The label is meaningful only for time comparisons
// (where it's "1d"/"1w"/"1m"); for version comparisons it's empty
// because ClientVersion already differs across the picks and serves
// as the chart split axis.
type pickedRun struct {
	run    BenchmarkRun
	bucket string
}

// manufactureGroup picks the source runs that belong in one synthetic
// group, deep-clones them with a rewritten BenchmarkRun ID and a
// prefixed TestName, and stamps TimeBucket so the frontend has a
// testConfig axis to split chart series on. Returns nil when
// fewer than two distinct buckets are available — a one-bucket
// "comparison" would just clutter the dropdown.
func manufactureGroup(sourceRuns []BenchmarkRun, networkSubstr, testName string, kind comparisonKind, now time.Time) []BenchmarkRun {
	var picked []pickedRun
	var bucketCount int
	var idLabel, prefix string

	switch kind {
	case compareByTime:
		picked, bucketCount = pickByTime(sourceRuns, now)
		idLabel = "time"
		prefix = "[Compare: Time]"
	case compareByVersion:
		picked, bucketCount = pickByVersion(sourceRuns)
		idLabel = "version"
		prefix = "[Compare: Versions]"
	}

	if bucketCount < 2 {
		return nil
	}

	syntheticID := fmt.Sprintf("compare-%s-%s", idLabel, slugify(networkSubstr+"-"+testName))
	// The dropdown label in the frontend shows "<testName> - <createdAt>"
	// for each run. For a synthetic group whose constituent runs span
	// multiple createdAts, picking any one of them is misleading
	// ("this comparison happened on 5/21" implies one moment in
	// time, but a Compare:Time group by definition spans many).
	// Stamp every clone's top-level createdAt to `now` so the
	// dropdown reads as the comparison-view freshness time rather
	// than an arbitrary source run's time. The actual per-run dates
	syntheticCreatedAt := now
	out := make([]BenchmarkRun, 0, len(picked))
	for _, p := range picked {
		clone := cloneBenchmarkRun(p.run)
		clone.TestConfig.BenchmarkRun = syntheticID
		if p.bucket != "" {
			clone.TestConfig.TimeBucket = p.bucket
		}
		clone.CreatedAt = &syntheticCreatedAt
		if !strings.HasPrefix(clone.TestName, prefix) {
			clone.TestName = prefix + " " + clone.TestName
		}
		out = append(out, clone)
	}
	return out
}

// variantKey identifies the (payload, gasLimit, nodeType, blockTime)
// combination of a run. Within one synthetic group we keep at most
// one source run per variant per bucket so the chart series stay
// comparable — otherwise two runs with the same variant would
// produce duplicate lines that obscure the difference being
// compared.
func variantKey(r BenchmarkRun) string {
	return fmt.Sprintf("%s|%d|%s|%d",
		r.TestConfig.TransactionPayload,
		r.TestConfig.GasLimit,
		r.TestConfig.NodeType,
		r.TestConfig.BlockTimeMilliseconds,
	)
}

// pickByTime selects the most-recent run per (variant, bucket) and
// pairs each pick with its bucket label so manufactureGroup can stamp
// the label into testConfig. The returned bucketCount counts how many
// buckets actually had data, so the caller can decide whether the
// result is a real comparison.
func pickByTime(runs []BenchmarkRun, now time.Time) ([]pickedRun, int) {
	latestPerKey := make([]map[string]BenchmarkRun, len(timeBuckets))
	for i := range latestPerKey {
		latestPerKey[i] = map[string]BenchmarkRun{}
	}
	for _, r := range runs {
		if r.CreatedAt == nil {
			continue
		}
		age := now.Sub(*r.CreatedAt)
		if age < 0 {
			age = 0
		}
		idx := -1
		for i, b := range timeBuckets {
			if age >= b.Newer && age < b.Older {
				idx = i
				break
			}
		}
		if idx == -1 {
			continue
		}
		key := variantKey(r)
		existing, ok := latestPerKey[idx][key]
		if !ok || r.CreatedAt.After(*existing.CreatedAt) {
			latestPerKey[idx][key] = r
		}
	}
	var out []pickedRun
	bucketCount := 0
	for i, m := range latestPerKey {
		if len(m) == 0 {
			continue
		}
		bucketCount++
		for _, r := range sortedByVariant(m) {
			out = append(out, pickedRun{run: r, bucket: timeBuckets[i].Label})
		}
	}
	return out, bucketCount
}

// pickByVersion selects the most-recent run per (variant, version).
// Returned bucketCount = number of distinct versions seen. The
// per-pick bucket label is left empty because the version itself
// (already in testConfig.ClientVersion) is the chart split axis.
func pickByVersion(runs []BenchmarkRun) ([]pickedRun, int) {
	byVersion := map[string]map[string]BenchmarkRun{}
	for _, r := range runs {
		v := versionOf(r)
		if v == "" || r.CreatedAt == nil {
			continue
		}
		bucket, ok := byVersion[v]
		if !ok {
			bucket = map[string]BenchmarkRun{}
			byVersion[v] = bucket
		}
		key := variantKey(r)
		existing, exists := bucket[key]
		if !exists || r.CreatedAt.After(*existing.CreatedAt) {
			bucket[key] = r
		}
	}
	var out []pickedRun
	versions := make([]string, 0, len(byVersion))
	for v := range byVersion {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	for _, v := range versions {
		for _, r := range sortedByVariant(byVersion[v]) {
			out = append(out, pickedRun{run: r})
		}
	}
	return out, len(byVersion)
}

// versionOf prefers TestConfig["ClientVersion"] (set by the future
// base/benchmark injection) and falls back to result.clientVersion.
// "" means "no version info" and the run is dropped from version
// comparisons rather than being lumped under an empty-string bucket.
func versionOf(r BenchmarkRun) string {
	if r.TestConfig.ClientVersion != "" {
		return r.TestConfig.ClientVersion
	}
	return r.Result.ClientVersion
}

func filterByNetwork(runs []BenchmarkRun, networkSubstr string) []BenchmarkRun {
	if networkSubstr == "" {
		return runs
	}
	needle := strings.ToLower(networkSubstr)
	out := make([]BenchmarkRun, 0, len(runs))
	for _, r := range runs {
		if strings.Contains(strings.ToLower(r.SourceFile), needle) {
			out = append(out, r)
		}
	}
	return out
}

// sortedByVariant is a small determinism guarantee — chart series
// order shouldn't depend on Go map iteration order, otherwise a
// browser refresh might shuffle the rows in the existing UI's
// per-gas-limit grouping.
func sortedByVariant(m map[string]BenchmarkRun) []BenchmarkRun {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]BenchmarkRun, 0, len(keys))
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}

// cloneBenchmarkRun returns a shallow-but-safe-for-our-mutations
// copy: the TestConfig struct is copied by value (all fields are
// scalar so no further work needed), and pointer-typed fields like
// CreatedAt are aliased — they're treated as immutable everywhere
// downstream, so sharing them is fine. If a downstream mutation ever
// touches CreatedAt, MachineInfo, or Thresholds, those fields must
// be deep-cloned here to avoid corrupting the source run.
func cloneBenchmarkRun(r BenchmarkRun) BenchmarkRun {
	return BenchmarkRun{
		ID:              r.ID,
		SourceFile:      r.SourceFile,
		OutputDir:       r.OutputDir,
		TestName:        r.TestName,
		TestDescription: r.TestDescription,
		TestConfig:      r.TestConfig,
		Result:          r.Result,
		Thresholds:      r.Thresholds,
		CreatedAt:       r.CreatedAt,
		BucketPath:      r.BucketPath,
		MachineInfo:     r.MachineInfo,
		ClientVersion:   r.ClientVersion,
	}
}

// slugify lowercases its input and replaces any non-alphanumeric
// run with a single dash, trimming leading/trailing dashes. Used to
// build stable, URL-safe synthetic IDs from network + testName.
func slugify(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevDash := true
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := b.String()
	return strings.Trim(out, "-")
}
