package services

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/ethereum/go-ethereum/log"
)

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
}

// S3Service handles interactions with AWS S3
type S3Service struct {
	client     *s3.S3
	bucketName string
	cache      *MemoryCache
	l          log.Logger
}

// NewS3Service creates a new S3 service instance
func NewS3Service(bucketName, region string, cache *MemoryCache, l log.Logger) (*S3Service, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("S3 bucket name is required")
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &S3Service{
		client:     s3.New(sess),
		bucketName: bucketName,
		cache:      cache,
		l:          l,
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

// GetMetadata retrieves and parses the metadata.json from S3
func (s *S3Service) GetMetadata() (*BenchmarkRuns, error) {
	data, err := s.GetObject("metadata.json")
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	var metadata BenchmarkRuns
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// GetMetrics retrieves metrics data for a specific run and node type
func (s *S3Service) GetMetrics(runID, outputDir, nodeType string) ([]byte, error) {
	key := fmt.Sprintf("%s/%s/metrics-%s.json", runID, outputDir, nodeType)
	return s.GetObject(key)
}
