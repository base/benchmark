package services

// BackendStorage is the interface both S3Service and LocalService
// implement. Handlers accept this interface so they work identically
// against a real S3 bucket or a local output directory.
//
// The local backend reads from the directory that `base-bench run
// --output-dir <dir>` writes — same layout as S3, so the server can
// be pointed straight at benchmark output without any S3 or MinIO
// setup.
type BackendStorage interface {
	GetMetadata() (*BenchmarkRuns, error)
	GetObject(key string) ([]byte, error)
	ListLoadTests(network string) ([]LoadTestEntry, error)
	GetLoadTest(network, timestamp string) ([]byte, error)
}
