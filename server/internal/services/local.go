package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// LocalService implements BackendStorage by reading from a local
// directory tree produced by `base-bench run --output-dir <dir>`.
// The layout matches the S3 layout the server expects in production:
//
//	<dir>/
//	├── <outputDir>/
//	│   ├── metadata.json          # this run's metadata (commit signal)
//	│   ├── metrics-sequencer.json # per-block sequencer timeseries
//	│   └── metrics-validator.json # per-block validator timeseries
//	└── load-tests/
//	    └── <network>/
//	        └── <timestamp>.json   # load test result
//
// LocalService uses the same merge/dedup/retention/synthesis pipeline
// as S3Service via the shared mergeRuns function, so reports rendered
// against local data look identical to production reports.
//
// Caching: LocalService uses file mtime as the fingerprint instead of
// ETag. The same metadataCache struct (and its TTL) is reused so the
// time.Now() correctness guarantee applies here too.
type LocalService struct {
	dir           string
	metadataCache metadataCache
	l             log.Logger
}

func NewLocalService(dir string, l log.Logger) (*LocalService, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("local dir %q: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local dir %q is not a directory", dir)
	}
	const defaultMaxFiles = 2000
	return &LocalService{
		dir: dir,
		metadataCache: metadataCache{
			files:     make(map[string][]byte, defaultMaxFiles),
			maxFiles:  defaultMaxFiles,
			mergedTTL: defaultMergedTTL,
		},
		l: l,
	}, nil
}

func (ls *LocalService) GetMetadata() (*BenchmarkRuns, error) {
	objects, err := ls.listLocalMetadataObjects()
	if err != nil {
		return nil, err
	}

	fingerprint := metadataObjectsFingerprint(objects)

	ls.metadataCache.mu.RLock()
	cacheValid := ls.metadataCache.cachedResult != nil &&
		ls.metadataCache.cachedKeyFingerprint == fingerprint &&
		time.Since(ls.metadataCache.cachedAt) < ls.metadataCache.mergedTTL
	if cacheValid {
		result := ls.metadataCache.cachedResult
		ls.metadataCache.mu.RUnlock()
		ls.l.Debug("Serving metadata from cache", "runs", len(result.Runs))
		return result, nil
	}
	ls.metadataCache.mu.RUnlock()

	ls.metadataCache.mu.Lock()
	defer ls.metadataCache.mu.Unlock()

	cacheValid = ls.metadataCache.cachedResult != nil &&
		ls.metadataCache.cachedKeyFingerprint == fingerprint &&
		time.Since(ls.metadataCache.cachedAt) < ls.metadataCache.mergedTTL
	if cacheValid {
		return ls.metadataCache.cachedResult, nil
	}

	var allRuns []BenchmarkRun
	fetched := 0
	cacheHits := 0
	for _, obj := range objects {
		fileKey := obj.key + "|" + obj.etag
		var data []byte
		if cached, ok := ls.metadataCache.files[fileKey]; ok {
			data = cached
			cacheHits++
		} else {
			var readErr error
			data, readErr = os.ReadFile(filepath.Join(ls.dir, obj.key))
			if readErr != nil {
				ls.l.Warn("Failed to read local metadata file, skipping", "key", obj.key, "error", readErr)
				continue
			}
			ls.metadataCache.setFile(fileKey, data)
			fetched++
		}

		var meta BenchmarkRuns
		if err := json.Unmarshal(data, &meta); err != nil {
			ls.l.Warn("Failed to parse local metadata file, skipping", "key", obj.key, "error", err)
			continue
		}
		allRuns = append(allRuns, meta.Runs...)
	}
	ls.l.Info("Local metadata file read complete", "fetched", fetched, "cacheHits", cacheHits)

	metadata := mergeRuns(allRuns, ls.l)
	ls.metadataCache.cachedResult = metadata
	ls.metadataCache.cachedKeyFingerprint = fingerprint
	ls.metadataCache.cachedAt = *metadata.CreatedAt
	ls.l.Info("Built merged metadata from local dir", "totalRuns", len(metadata.Runs), "dir", ls.dir)
	return metadata, nil
}

func (ls *LocalService) GetObject(key string) ([]byte, error) {
	path, err := ls.safePath(key)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func (ls *LocalService) ListLoadTests(network string) ([]LoadTestEntry, error) {
	dir, err := ls.safePath(filepath.Join("load-tests", network))
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("listing load tests for %s: %w", network, err)
	}
	var result []LoadTestEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		result = append(result, LoadTestEntry{
			Network:   network,
			Timestamp: strings.TrimSuffix(e.Name(), ".json"),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp > result[j].Timestamp
	})
	return result, nil
}

func (ls *LocalService) GetLoadTest(network, timestamp string) ([]byte, error) {
	path, err := ls.safePath(filepath.Join("load-tests", network, timestamp+".json"))
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// safePath resolves rel against ls.dir and returns an error if the
// result escapes ls.dir, preventing path-traversal attacks when rel
// contains user-provided values like HTTP path parameters.
func (ls *LocalService) safePath(rel string) (string, error) {
	abs := filepath.Clean(filepath.Join(ls.dir, rel))
	base := filepath.Clean(ls.dir) + string(filepath.Separator)
	if abs != filepath.Clean(ls.dir) && !strings.HasPrefix(abs, base) {
		return "", fmt.Errorf("path %q escapes root directory", rel)
	}
	return abs, nil
}

// listLocalMetadataObjects walks the local directory for
// <outputDir>/metadata.json files and returns them as metadataObjects.
// The etag field is populated from the file's mtime (nanoseconds as a
// string) so the same content-fingerprint invalidation logic applies
// — a newly written metadata.json triggers a cache miss on the next
// request.
func (ls *LocalService) listLocalMetadataObjects() ([]metadataObject, error) {
	var objects []metadataObject
	entries, err := os.ReadDir(ls.dir)
	if err != nil {
		return nil, fmt.Errorf("reading local dir %q: %w", ls.dir, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if nonRunPrefixes[e.Name()+"/"] {
			continue
		}
		metaPath := filepath.Join(ls.dir, e.Name(), "metadata.json")
		info, err := os.Stat(metaPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			ls.l.Warn("Failed to stat local metadata file", "path", metaPath, "error", err)
			continue
		}
		key := e.Name() + "/metadata.json"
		etag := fmt.Sprintf("%d", info.ModTime().UnixNano())
		objects = append(objects, metadataObject{key: key, etag: etag})
	}
	return objects, nil
}
