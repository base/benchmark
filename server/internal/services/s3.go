package services

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/ethereum/go-ethereum/log"
)

const defaultMergedTTL = time.Hour

// BenchmarkRuns represents the metadata structure
type BenchmarkRuns struct {
	Runs      []BenchmarkRun `json:"runs"`
	CreatedAt *time.Time     `json:"createdAt"`
}

type BenchmarkTestConfig struct {
	BenchmarkRun          string `json:"BenchmarkRun"`
	BlockTimeMilliseconds int    `json:"BlockTimeMilliseconds"`
	GasLimit              int    `json:"GasLimit"`
	NodeType              string `json:"NodeType"`
	TransactionPayload    string `json:"TransactionPayload"`
	// ClientVersion is populated by base/benchmark when it learns
	// the EL binary's version (via web3_clientVersion or the
	// BASE_BENCH_CLIENT_VERSION env override). Empty for runs
	// produced before that injection landed.
	ClientVersion string `json:"ClientVersion,omitempty"`
	// TimeBucket is stamped onto synthetic comparison-group clones
	// by services/comparison.go so the frontend's auto-discovered
	// "Show Line Per" dropdown has a meaningful axis to split time
	// comparisons on ("1d" / "1w" / "1m"). Always absent on natural
	// per-run pages — the omitempty tag ensures it doesn't appear in
	// the JSON for source runs that never had it.
	TimeBucket string `json:"TimeBucket,omitempty"`
}

type SequencerMetrics struct {
	GasPerSecond      float64 `json:"gasPerSecond"`
	ForkChoiceUpdated float64 `json:"forkChoiceUpdated"`
	GetPayload        float64 `json:"getPayload"`
	SendTxs           float64 `json:"sendTxs"`
}

type ValidatorMetrics struct {
	GasPerSecond float64 `json:"gasPerSecond"`
	NewPayload   float64 `json:"newPayload"`
}

type BenchmarkResult struct {
	Success          bool             `json:"success"`
	Complete         bool             `json:"complete"`
	SequencerMetrics SequencerMetrics `json:"sequencerMetrics"`
	ValidatorMetrics ValidatorMetrics `json:"validatorMetrics"`
	ClientVersion    string           `json:"clientVersion,omitempty"`
}

type MachineInfo struct {
	Type       string `json:"type"`
	Provider   string `json:"provider"`
	Region     string `json:"region"`
	FileSystem string `json:"fileSystem"`
}

// BenchmarkRun represents a single benchmark run
type BenchmarkRun struct {
	ID              string              `json:"id"`
	SourceFile      string              `json:"sourceFile"`
	OutputDir       string              `json:"outputDir"`
	TestName        string              `json:"testName"`
	TestDescription string              `json:"testDescription"`
	TestConfig      BenchmarkTestConfig `json:"testConfig"`
	Result          BenchmarkResult     `json:"result"`
	Thresholds      interface{}         `json:"thresholds"`
	CreatedAt       *time.Time          `json:"createdAt"`
	BucketPath      string              `json:"bucketPath,omitempty"`
	MachineInfo     MachineInfo         `json:"machineInfo,omitempty"`
	ClientVersion   string              `json:"clientVersion,omitempty"`
}

// metadataObject is one <outputDir>/metadata.json object as returned
// by the S3 listing. etag is the object's ETag; it changes whenever
// the file is overwritten, so it serves as the per-file identity for
// cache invalidation.
type metadataObject struct {
	key  string
	etag string
}

// metadataCache holds the merged result and a per-file byte cache.
//
// Per-file cache (files): stores raw bytes keyed on "key|etag" so a
// rewritten metadata.json (same key, new ETag) is treated as a cache
// miss and re-fetched. Bounded at maxFiles entries (~3 MB at ~1.5 KB
// each) with FIFO eviction.
//
// Merged result (cachedResult): stores the fully-processed
// *BenchmarkRuns. Fingerprint-invalidated on new/changed objects AND
// time-expired after mergedTTL because the result depends on
// time.Now() through applyRetentionPolicy and
// synthesizeComparisonGroups (1d/1w/1m buckets). Without the TTL,
// the comparison buckets would be pinned to the first rebuild
// forever if no new runs land.
//
// mu protects all fields. RLock on the hot path, Lock for rebuilds.
type metadataCache struct {
	mu                   sync.RWMutex
	cachedResult         *BenchmarkRuns
	cachedKeyFingerprint string
	cachedAt             time.Time
	mergedTTL            time.Duration
	files                map[string][]byte
	fileOrder            []string
	maxFiles             int
}

type S3Service struct {
	client        *s3.S3
	bucketName    string
	cache         *MemoryCache
	metadataCache metadataCache
	l             log.Logger
}

// NewS3Service creates a new S3 service instance.
//
// endpoint is optional. When non-empty, it overrides the default AWS S3
// endpoint and forces path-style addressing — required for MinIO and
// other S3-compatible stores. Production AWS deployments leave it empty.
func NewS3Service(bucketName, region, endpoint string, cache *MemoryCache, l log.Logger) (*S3Service, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("S3 bucket name is required")
	}

	awsCfg := &aws.Config{
		Region: aws.String(region),
	}
	if endpoint != "" {
		awsCfg.Endpoint = aws.String(endpoint)
		awsCfg.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession(awsCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	const defaultMaxMetadataFiles = 2000

	return &S3Service{
		client:     s3.New(sess),
		bucketName: bucketName,
		cache:      cache,
		metadataCache: metadataCache{
			files:     make(map[string][]byte, defaultMaxMetadataFiles),
			maxFiles:  defaultMaxMetadataFiles,
			mergedTTL: defaultMergedTTL,
		},
		l: l,
	}, nil
}

// GetObject retrieves an object from S3 with caching
func (s *S3Service) GetObject(key string) ([]byte, error) {
	// Check cache first if available
	if s.cache != nil {
		if cached, hit := s.cache.Get(key); hit {
			s.l.Debug("Cache hit", "key", key)
			return cached, nil
		}
	}

	s.l.Debug("Fetching from S3", "key", key, "bucket", s.bucketName)

	result, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %w", key, err)
	}
	defer result.Body.Close()

	// Read the entire body
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	// Cache the result if cache is available
	if s.cache != nil {
		s.cache.Set(key, data)
	}

	return data, nil
}

// GetMetadata returns the merged benchmark metadata. Results are
// cached by the content fingerprint of the actual metadata.json S3
// objects (key + ETag), not just the prefix list. This means:
//   - A new run (new key) invalidates the fingerprint immediately.
//   - An overwritten metadata.json (new ETag on the same key) also
//     invalidates, so resubmitted or corrected runs are never missed.
//   - The merged result also expires after mergedTTL (default 1h)
//     because it depends on time.Now() via applyRetentionPolicy and
//     synthesizeComparisonGroups.
func (s *S3Service) GetMetadata() (*BenchmarkRuns, error) {
	objects, err := s.listMetadataObjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list metadata files: %w", err)
	}

	fingerprint := metadataObjectsFingerprint(objects)

	s.metadataCache.mu.RLock()
	cacheValid := s.metadataCache.cachedResult != nil &&
		s.metadataCache.cachedKeyFingerprint == fingerprint &&
		time.Since(s.metadataCache.cachedAt) < s.metadataCache.mergedTTL
	if cacheValid {
		result := s.metadataCache.cachedResult
		s.metadataCache.mu.RUnlock()
		s.l.Debug("Serving metadata from cache", "runs", len(result.Runs))
		return result, nil
	}
	s.metadataCache.mu.RUnlock()

	s.l.Info("Rebuilding metadata from S3", "objects", len(objects))

	s.metadataCache.mu.Lock()
	defer s.metadataCache.mu.Unlock()

	// Re-check under the write lock — another goroutine may have
	// rebuilt while we waited to acquire it.
	cacheValid = s.metadataCache.cachedResult != nil &&
		s.metadataCache.cachedKeyFingerprint == fingerprint &&
		time.Since(s.metadataCache.cachedAt) < s.metadataCache.mergedTTL
	if cacheValid {
		return s.metadataCache.cachedResult, nil
	}

	s.l.Info("Found per-run metadata files", "count", len(objects))

	var allRuns []BenchmarkRun
	fetched := 0
	cacheHits := 0
	for _, obj := range objects {
		fileKey := obj.key + "|" + obj.etag
		var data []byte
		if cached, ok := s.metadataCache.files[fileKey]; ok {
			data = cached
			cacheHits++
		} else {
			var fetchErr error
			data, fetchErr = s.getObjectDirect(obj.key)
			if fetchErr != nil {
				s.l.Warn("Failed to fetch metadata file, skipping", "key", obj.key, "error", fetchErr)
				continue
			}
			s.metadataCache.setFile(fileKey, data)
			fetched++
		}

		var meta BenchmarkRuns
		if err := json.Unmarshal(data, &meta); err != nil {
			s.l.Warn("Failed to parse metadata file, skipping", "key", obj.key, "error", err)
			continue
		}

		allRuns = append(allRuns, meta.Runs...)
	}
	s.l.Info("Metadata file fetch complete", "fetched", fetched, "cacheHits", cacheHits)

	metadata := mergeRuns(allRuns, s.l)

	s.metadataCache.cachedResult = metadata
	s.metadataCache.cachedKeyFingerprint = fingerprint
	s.metadataCache.cachedAt = *metadata.CreatedAt

	s.l.Info("Built merged metadata", "totalRuns", len(metadata.Runs), "metadataFiles", len(objects))
	return metadata, nil
}

// setFile stores raw bytes under the given cache key (key|etag).
// When at capacity, the oldest-inserted entry is evicted.
func (mc *metadataCache) setFile(key string, data []byte) {
	if _, exists := mc.files[key]; !exists {
		if len(mc.files) >= mc.maxFiles && len(mc.fileOrder) > 0 {
			oldest := mc.fileOrder[0]
			mc.fileOrder = mc.fileOrder[1:]
			delete(mc.files, oldest)
		}
		mc.fileOrder = append(mc.fileOrder, key)
	}
	mc.files[key] = data
}

// nonRunPrefixes are top-level S3 prefixes that don't represent
// benchmark runs and must be excluded from the metadata listing.
// "metadata/" is the legacy pre-migration location for the central
// metadata files (kept here defensively so it's ignored even if some
// legacy data isn't cleaned up); "load-tests/" is the load-test
// storage served by a separate handler.
var nonRunPrefixes = map[string]bool{
	"metadata/":   true,
	"load-tests/": true,
}

// listMetadataObjects lists every <outputDir>/metadata.json object
// that actually exists in the bucket, along with its ETag. Only
// existing objects are returned — in-progress runs whose
// metadata.json hasn't landed yet are absent from the list and
// therefore invisible to the merger (this is the commit-signal
// property of the per-run layout). Prefixes in nonRunPrefixes are
// skipped.
func (s *S3Service) listMetadataObjects() ([]metadataObject, error) {
	var objects []metadataObject

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
	}

	err := s.client.ListObjectsV2Pages(input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			if obj.Key == nil || !strings.HasSuffix(*obj.Key, "/metadata.json") {
				continue
			}
			key := *obj.Key
			prefix := key[:strings.Index(key, "/")+1]
			if nonRunPrefixes[prefix] {
				continue
			}
			etag := ""
			if obj.ETag != nil {
				etag = strings.Trim(*obj.ETag, `"`)
			}
			objects = append(objects, metadataObject{key: key, etag: etag})
		}
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list metadata objects: %w", err)
	}

	return objects, nil
}

// metadataObjectsFingerprint returns a string that changes whenever
// the set of metadata.json objects changes — either a new key
// appearing or an existing key's ETag changing (file overwritten).
func metadataObjectsFingerprint(objects []metadataObject) string {
	parts := make([]string, len(objects))
	for i, o := range objects {
		parts[i] = o.key + "|" + o.etag
	}
	sort.Strings(parts)
	return strings.Join(parts, "\n")
}

// getObjectDirect fetches an S3 object without caching (used for individual metadata files)
func (s *S3Service) getObjectDirect(key string) ([]byte, error) {
	result, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %w", key, err)
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

// applyRetentionPolicy filters runs according to the retention policy:
// - Keep all runs from the past 2 weeks
// - Keep one run per month for the past 6 months (the run closest to the 1st of each month)
// - Drop everything older
// Retained monthly runs get their TestName prefixed with the month label.
func applyRetentionPolicy(runs []BenchmarkRun, l log.Logger) []BenchmarkRun {
	now := time.Now()
	recentCutoff := now.AddDate(0, 0, -14) // 2 weeks ago
	monthlyCutoff := now.AddDate(0, -6, 0) // 6 months ago

	var recentRuns []BenchmarkRun

	type monthCandidate struct {
		run      BenchmarkRun
		distance time.Duration
	}
	monthBuckets := make(map[string]*monthCandidate)

	for _, run := range runs {
		if run.CreatedAt == nil {
			l.Debug("Dropping run with nil CreatedAt during retention", "runID", run.ID)
			continue
		}

		if !run.CreatedAt.Before(recentCutoff) {
			recentRuns = append(recentRuns, run)
			continue
		}

		if run.CreatedAt.Before(monthlyCutoff) {
			l.Debug("Dropping run outside retention window", "runID", run.ID, "createdAt", run.CreatedAt)
			continue
		}

		// Between 2 weeks and 6 months — pick one per month (closest to 1st of month)
		monthKey := run.CreatedAt.Format("2006-01")
		firstOfMonth := time.Date(run.CreatedAt.Year(), run.CreatedAt.Month(), 1, 0, 0, 0, 0, run.CreatedAt.Location())
		dist := run.CreatedAt.Sub(firstOfMonth)
		if dist < 0 {
			dist = -dist
		}

		existing, ok := monthBuckets[monthKey]
		if !ok || dist < existing.distance {
			monthBuckets[monthKey] = &monthCandidate{run: run, distance: dist}
		}
	}

	// Build the monthly runs with prefixed TestName
	var monthlyRuns []BenchmarkRun
	for monthKey, candidate := range monthBuckets {
		run := candidate.run
		t, _ := time.Parse("2006-01", monthKey)
		label := t.Format("Jan 2006")
		prefix := fmt.Sprintf("[Monthly - %s] ", label)
		if !strings.HasPrefix(run.TestName, "[Monthly") {
			run.TestName = prefix + run.TestName
		}
		monthlyRuns = append(monthlyRuns, run)
	}

	// Sort monthly runs chronologically
	sort.Slice(monthlyRuns, func(i, j int) bool {
		return monthlyRuns[i].CreatedAt.Before(*monthlyRuns[j].CreatedAt)
	})

	result := append(monthlyRuns, recentRuns...)
	l.Info("Applied retention policy", "input", len(runs), "kept", len(result),
		"recent", len(recentRuns), "monthly", len(monthlyRuns))
	return result
}

// mergeRuns deduplicates, sorts, applies retention, and synthesizes
// comparison groups from a flat list of parsed BenchmarkRun values.
// It is the shared pipeline used by both S3Service and LocalService.
func mergeRuns(allRuns []BenchmarkRun, l log.Logger) *BenchmarkRuns {
	seen := make(map[string]int)
	var deduped []BenchmarkRun
	for _, run := range allRuns {
		key := run.ID + "|" + run.OutputDir
		if idx, exists := seen[key]; exists {
			deduped[idx] = run
		} else {
			seen[key] = len(deduped)
			deduped = append(deduped, run)
		}
	}

	sort.Slice(deduped, func(i, j int) bool {
		if deduped[i].CreatedAt == nil {
			return true
		}
		if deduped[j].CreatedAt == nil {
			return false
		}
		return deduped[i].CreatedAt.Before(*deduped[j].CreatedAt)
	})

	deduped = applyRetentionPolicy(deduped, l)

	now := time.Now()
	deduped = append(deduped, synthesizeComparisonGroups(deduped, now)...)
	return &BenchmarkRuns{Runs: deduped, CreatedAt: &now}
}

// GetMetrics retrieves metrics data for a specific run and node type.
// The S3 key is <outputDir>/metrics-<nodeType>.json since run files are
// uploaded flat under their outputDir prefix.
func (s *S3Service) GetMetrics(outputDir, nodeType string) ([]byte, error) {
	key := fmt.Sprintf("%s/metrics-%s.json", outputDir, nodeType)
	return s.GetObject(key)
}

// LoadTestEntry represents a single load test run stored in S3.
type LoadTestEntry struct {
	Network   string `json:"network"`
	Timestamp string `json:"timestamp"`
}

// ListLoadTests lists all load test result timestamps for a given network,
// ordered newest-first.
func (s *S3Service) ListLoadTests(network string) ([]LoadTestEntry, error) {
	prefix := fmt.Sprintf("load-tests/%s/", network)
	var entries []LoadTestEntry

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	}

	err := s.client.ListObjectsV2Pages(input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			if obj.Key == nil || !strings.HasSuffix(*obj.Key, ".json") {
				continue
			}
			parts := strings.Split(*obj.Key, "/")
			filename := parts[len(parts)-1]
			timestamp := strings.TrimSuffix(filename, ".json")
			entries = append(entries, LoadTestEntry{
				Network:   network,
				Timestamp: timestamp,
			})
		}
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list load test results: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp > entries[j].Timestamp
	})

	return entries, nil
}

// GetLoadTest fetches a single load test result JSON from S3.
func (s *S3Service) GetLoadTest(network, timestamp string) ([]byte, error) {
	key := fmt.Sprintf("load-tests/%s/%s.json", network, timestamp)
	return s.GetObject(key)
}
