package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/base/base-bench/runner/benchmark"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// S3Service handles interactions with AWS S3 for benchmark data
type S3Service struct {
	client     *s3.S3
	bucketName string
	log        log.Logger
}

// NewS3Service creates a new S3 service instance
func NewS3Service(bucketName string, log log.Logger) (*S3Service, error) {
	if bucketName == "" {
		return nil, errors.New("bucket name is required")
	}

	// Create AWS session with default credentials
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"), // Default region as per spec
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AWS session")
	}

	return &S3Service{
		client:     s3.New(sess),
		bucketName: bucketName,
		log:        log,
	}, nil
}

// UploadRunResults uploads all files from a test run output directory to S3
func (s *S3Service) UploadRunResults(outputDir, runID, runOutputDir string) (string, error) {
	s.log.Info("Uploading run results to S3", "outputDir", outputDir, "runID", runID, "runOutputDir", runOutputDir, "bucket", s.bucketName)

	// Create S3 key prefix for this run using id/outputDir structure
	keyPrefix := fmt.Sprintf("%s/%s/", runID, runOutputDir)

	// Walk through the output directory and upload all files
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from output directory
		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			return errors.Wrap(err, "failed to get relative path")
		}

		// Create S3 key
		s3Key := keyPrefix + strings.ReplaceAll(relPath, "\\", "/")

		// Upload file
		err = s.uploadFile(path, s3Key)
		if err != nil {
			return errors.Wrapf(err, "failed to upload file %s", path)
		}

		s.log.Info("Uploaded file to S3", "localPath", path, "s3Key", s3Key)
		return nil
	})

	if err != nil {
		return "", errors.Wrap(err, "failed to upload run results")
	}

	return keyPrefix, nil
}

// uploadFile uploads a single file to S3
func (s *S3Service) uploadFile(localPath, s3Key string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer file.Close()

	_, err = s.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
		Body:   file,
	})

	return errors.Wrap(err, "failed to upload to S3")
}

// SyncMetadata downloads the current metadata.json from S3, merges the new run, and uploads back
func (s *S3Service) SyncMetadata(newRun benchmark.Run) error {
	s.log.Info("Syncing metadata with S3", "runID", newRun.ID, "bucket", s.bucketName)

	// Download existing metadata from S3
	existingMetadata, err := s.downloadMetadata()
	if err != nil && !isNoSuchKeyError(err) {
		return errors.Wrap(err, "failed to download existing metadata")
	}

	// If no existing metadata, create new one
	if existingMetadata == nil {
		now := time.Now()
		existingMetadata = &benchmark.RunGroup{
			Runs:      []benchmark.Run{},
			CreatedAt: &now,
		}
	}

	// Create a temporary local metadata with just the new run for merging
	localMetadata := &benchmark.RunGroup{
		Runs:      []benchmark.Run{newRun},
		CreatedAt: existingMetadata.CreatedAt,
	}

	// Use chronological merge to ensure proper ordering
	mergedMetadata := s.mergeMetadataChronologically(localMetadata, existingMetadata)

	// Upload updated metadata back to S3
	err = s.uploadMetadata(mergedMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to upload updated metadata")
	}

	s.log.Info("Successfully synced metadata with S3", "runID", newRun.ID, "totalRuns", len(mergedMetadata.Runs))
	return nil
}

// downloadMetadata downloads the metadata.json file from S3
func (s *S3Service) downloadMetadata() (*benchmark.RunGroup, error) {
	result, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String("metadata.json"),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read metadata from S3")
	}

	var metadata benchmark.RunGroup
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal metadata")
	}

	return &metadata, nil
}

// uploadMetadata uploads the metadata.json file to S3
func (s *S3Service) uploadMetadata(metadata *benchmark.RunGroup) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal metadata")
	}

	_, err = s.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String("metadata.json"),
		Body:   bytes.NewReader(data),
	})

	return errors.Wrap(err, "failed to upload metadata to S3")
}

// ExportOutputDirectory exports an entire output directory to S3 (for standalone export command)
func (s *S3Service) ExportOutputDirectory(outputDir string) error {
	s.log.Info("Exporting entire output directory to S3", "outputDir", outputDir, "bucket", s.bucketName)

	// Load local metadata.json
	metadataPath := filepath.Join(outputDir, "metadata.json")
	localMetadata, err := s.loadLocalMetadata(metadataPath)
	if err != nil {
		return errors.Wrap(err, "failed to load local metadata")
	}

	// Download existing S3 metadata
	s3Metadata, err := s.downloadMetadata()
	if err != nil && !isNoSuchKeyError(err) {
		return errors.Wrap(err, "failed to download S3 metadata")
	}

	// If no S3 metadata exists, start with empty one
	if s3Metadata == nil {
		now := time.Now()
		s3Metadata = &benchmark.RunGroup{
			Runs:      []benchmark.Run{},
			CreatedAt: &now,
		}
	}

	// Upload each run's output directory and update metadata
	for i := range localMetadata.Runs {
		run := &localMetadata.Runs[i]
		runOutputDir := filepath.Join(outputDir, run.OutputDir)

		// Check if run output directory exists
		if _, err := os.Stat(runOutputDir); os.IsNotExist(err) {
			s.log.Warn("Run output directory does not exist, skipping", "runID", run.ID, "outputDir", runOutputDir)
			continue
		}

		// Upload run results and get bucket path - now includes outputDir in the path
		bucketPath, err := s.UploadRunResults(runOutputDir, run.ID, run.OutputDir)
		if err != nil {
			return errors.Wrapf(err, "failed to upload run results for %s", run.ID)
		}

		// Update run with bucket path
		run.BucketPath = bucketPath
	}

	// Merge local and S3 metadata in chronological order
	mergedMetadata := s.mergeMetadataChronologically(localMetadata, s3Metadata)

	// Upload updated metadata to S3
	err = s.uploadMetadata(mergedMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to upload final metadata to S3")
	}

	s.log.Info("Successfully exported output directory to S3", "localRuns", len(localMetadata.Runs), "totalMergedRuns", len(mergedMetadata.Runs))
	return nil
}

// mergeMetadataChronologically merges local and cloud metadata, ensuring runs are ordered chronologically by createdAt
func (s *S3Service) mergeMetadataChronologically(localMetadata, cloudMetadata *benchmark.RunGroup) *benchmark.RunGroup {
	// Create a map to track existing runs by compound key (runID + outputDir) to avoid duplicates
	existingRuns := make(map[string]*benchmark.Run)
	var allRuns []benchmark.Run

	// Helper function to create compound key
	getCompoundKey := func(run *benchmark.Run) string {
		return run.ID + "|" + run.OutputDir
	}

	// First, add all cloud metadata runs to avoid duplicates
	for i := range cloudMetadata.Runs {
		run := &cloudMetadata.Runs[i]
		compoundKey := getCompoundKey(run)
		existingRuns[compoundKey] = run
		allRuns = append(allRuns, *run)
	}

	// Then add local runs, replacing existing ones or adding new ones
	for i := range localMetadata.Runs {
		run := &localMetadata.Runs[i]
		compoundKey := getCompoundKey(run)

		if _, exists := existingRuns[compoundKey]; exists {
			// Replace existing run with local version (local is more up-to-date)
			for j := range allRuns {
				if getCompoundKey(&allRuns[j]) == compoundKey {
					allRuns[j] = *run
					break
				}
			}
			s.log.Info("Updated existing run", "runID", run.ID, "outputDir", run.OutputDir)
		} else {
			// Add new run
			allRuns = append(allRuns, *run)
			existingRuns[compoundKey] = run
			s.log.Info("Added new run", "runID", run.ID, "outputDir", run.OutputDir)
		}
	}

	// Sort all runs chronologically by createdAt
	sort.Slice(allRuns, func(i, j int) bool {
		// Handle nil createdAt values by treating them as very old
		if allRuns[i].CreatedAt == nil && allRuns[j].CreatedAt == nil {
			return false // maintain relative order if both are nil
		}
		if allRuns[i].CreatedAt == nil {
			return true // nil comes first (older)
		}
		if allRuns[j].CreatedAt == nil {
			return false // non-nil comes after nil
		}
		return allRuns[i].CreatedAt.Before(*allRuns[j].CreatedAt)
	})

	// Use the most recent createdAt from cloudMetadata, or current time if none exists
	mergedCreatedAt := cloudMetadata.CreatedAt
	if mergedCreatedAt == nil {
		now := time.Now()
		mergedCreatedAt = &now
	}

	s.log.Info("Merged metadata chronologically", "totalRuns", len(allRuns), "cloudRuns", len(cloudMetadata.Runs), "localRuns", len(localMetadata.Runs))

	return &benchmark.RunGroup{
		Runs:      allRuns,
		CreatedAt: mergedCreatedAt,
	}
}

// loadLocalMetadata loads metadata from a local file
func (s *S3Service) loadLocalMetadata(metadataPath string) (*benchmark.RunGroup, error) {
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read local metadata file")
	}

	var metadata benchmark.RunGroup
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal local metadata")
	}

	return &metadata, nil
}

// isNoSuchKeyError checks if the error is a "NoSuchKey" error from S3
func isNoSuchKeyError(err error) bool {
	return strings.Contains(err.Error(), "NoSuchKey")
}
