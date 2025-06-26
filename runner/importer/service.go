package importer

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/base/base-bench/benchmark/config"
	"github.com/base/base-bench/runner/benchmark"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// Service handles importing benchmark runs from files or URLs
type Service struct {
	config *config.ImportCmdConfig
	log    log.Logger
}

// NewService creates a new import service
func NewService(cfg *config.ImportCmdConfig, log log.Logger) *Service {
	return &Service{
		config: cfg,
		log:    log,
	}
}

// downloadFile downloads a file from a URL to a local path
func (s *Service) downloadFile(fileURL, localPath string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return errors.Wrap(err, "failed to create directory")
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		return errors.Wrap(err, "failed to fetch file")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return errors.Wrap(err, "failed to create local file")
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to copy file content")
	}

	return nil
}

// downloadOutputFiles downloads all output files for a run from a base URL
func (s *Service) downloadOutputFiles(baseURL, runOutputDir string) error {
	// List of expected files in output directories
	expectedFiles := []string{
		"logs-validator.gz",
		"result-validator.json",
		"logs-sequencer.gz",
		"result-sequencer.json",
		"metrics-validator.json",
		"metrics-sequencer.json",
	}

	localOutputDir := filepath.Join(s.config.OutputDir(), runOutputDir)
	s.log.Info("Downloading output files", "runOutputDir", runOutputDir, "localPath", localOutputDir)

	downloadedCount := 0
	for _, fileName := range expectedFiles {
		// Construct the URL for this file
		fileURL := baseURL + "/" + runOutputDir + "/" + fileName
		localFilePath := filepath.Join(localOutputDir, fileName)

		// Try to download the file
		err := s.downloadFile(fileURL, localFilePath)
		if err != nil {
			s.log.Warn("Failed to download file (continuing)", "file", fileName, "url", fileURL, "error", err)
		} else {
			s.log.Debug("Downloaded file", "file", fileName, "localPath", localFilePath)
			downloadedCount++
		}
	}

	s.log.Info("Downloaded output files", "runOutputDir", runOutputDir, "downloaded", downloadedCount, "total", len(expectedFiles))
	return nil
}

// LoadSourceMetadata loads metadata from a file or URL
func (s *Service) LoadSourceMetadata(source string) (*benchmark.RunGroup, error) {
	s.log.Info("Loading source metadata", "source", source)

	var reader io.Reader
	var baseURL string

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// Load from URL
		resp, err := http.Get(source)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch metadata from URL")
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
		}

		reader = resp.Body

		// Extract base URL for downloading output files
		u, err := url.Parse(source)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse source URL")
		}
		// Remove the filename to get the base directory URL
		u.Path = filepath.Dir(u.Path)
		baseURL = u.String()
	} else {
		// Load from file
		file, err := os.Open(source)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open source metadata file")
		}
		defer func() { _ = file.Close() }()

		reader = file
	}

	var metadata benchmark.RunGroup
	if err := json.NewDecoder(reader).Decode(&metadata); err != nil {
		return nil, errors.Wrap(err, "failed to decode source metadata JSON")
	}

	s.log.Info("Loaded source metadata", "runs", len(metadata.Runs))

	// If we loaded from URL, download output files for each run
	if baseURL != "" {
		s.log.Info("Downloading output files for all runs", "baseURL", baseURL)
		for _, run := range metadata.Runs {
			if run.OutputDir != "" {
				err := s.downloadOutputFiles(baseURL, run.OutputDir)
				if err != nil {
					s.log.Warn("Failed to download output files for run", "runID", run.ID, "outputDir", run.OutputDir, "error", err)
					// Continue with other runs even if one fails
				}
			}
		}
	}

	return &metadata, nil
}

// LoadDestinationMetadata loads the existing destination metadata
func (s *Service) LoadDestinationMetadata() (*benchmark.RunGroup, error) {
	metadataPath := path.Join(s.config.OutputDir(), "metadata.json")
	s.log.Info("Loading destination metadata", "path", metadataPath)

	file, err := os.Open(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, return empty metadata
			s.log.Info("Destination metadata file does not exist, creating new one")
			return &benchmark.RunGroup{Runs: []benchmark.Run{}}, nil
		}
		return nil, errors.Wrap(err, "failed to open destination metadata file")
	}
	defer func() { _ = file.Close() }()

	var metadata benchmark.RunGroup
	if err := json.NewDecoder(file).Decode(&metadata); err != nil {
		return nil, errors.Wrap(err, "failed to decode destination metadata JSON")
	}

	s.log.Info("Loaded destination metadata", "runs", len(metadata.Runs))
	return &metadata, nil
}

// generateRandomID generates a random ID for BenchmarkRun
func (s *Service) generateRandomID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", errors.Wrap(err, "failed to generate random ID")
	}
	return hex.EncodeToString(bytes), nil
}

// getLastBenchmarkRunID gets the BenchmarkRun ID from the run with the latest CreatedAt timestamp
func (s *Service) getLastBenchmarkRunID(destMetadata *benchmark.RunGroup) string {
	if len(destMetadata.Runs) == 0 {
		return ""
	}

	var latestRun *benchmark.Run
	var latestTime *time.Time

	// Find the run with the latest CreatedAt timestamp
	for i := range destMetadata.Runs {
		run := &destMetadata.Runs[i]
		if run.CreatedAt != nil {
			if latestTime == nil || run.CreatedAt.After(*latestTime) {
				latestTime = run.CreatedAt
				latestRun = run
			}
		}
	}

	// If no run had a CreatedAt, fall back to the last run in the array
	if latestRun == nil {
		latestRun = &destMetadata.Runs[len(destMetadata.Runs)-1]
	}

	// Extract BenchmarkRun ID from the latest run
	if latestRun.TestConfig != nil {
		if benchmarkRun, ok := latestRun.TestConfig[benchmark.BenchmarkRunTag]; ok {
			if benchmarkRunStr, ok := benchmarkRun.(string); ok {
				s.log.Debug("Found latest BenchmarkRun ID", "runID", latestRun.ID, "benchmarkRunID", benchmarkRunStr, "createdAt", latestRun.CreatedAt)
				return benchmarkRunStr
			}
		}
	}

	return ""
}

// fillCreatedAt fills missing CreatedAt fields using the root metadata CreatedAt or current time
func (s *Service) fillCreatedAt(runs []benchmark.Run, rootCreatedAt *time.Time) []benchmark.Run {
	now := time.Now()
	fallbackTime := &now

	// Use root CreatedAt if available, otherwise current time
	if rootCreatedAt != nil {
		fallbackTime = rootCreatedAt
	}

	for i := range runs {
		if runs[i].CreatedAt == nil {
			runs[i].CreatedAt = fallbackTime
			s.log.Debug("Filled missing CreatedAt for run", "runID", runs[i].ID, "createdAt", fallbackTime)
		}
	}

	return runs
}

// markRunsAsComplete sets the complete field to true for all imported runs
func (s *Service) markRunsAsComplete(runs []benchmark.Run) []benchmark.Run {
	for i := range runs {
		// Initialize Result if it doesn't exist
		if runs[i].Result == nil {
			runs[i].Result = &benchmark.RunResult{}
		}

		// Set complete to true
		runs[i].Result.Complete = true
		s.log.Debug("Marked run as complete", "runID", runs[i].ID)
	}

	return runs
}

// ApplyTags applies tags to runs in the metadata
func (s *Service) ApplyTags(runs []benchmark.Run, tag *config.TagConfig) []benchmark.Run {
	if tag == nil {
		return runs
	}

	s.log.Info("Applying tag", "key", tag.Key, "value", tag.Value, "runs", len(runs))

	for i := range runs {
		if runs[i].TestConfig == nil {
			runs[i].TestConfig = make(map[string]interface{})
		}
		runs[i].TestConfig[tag.Key] = tag.Value
	}

	return runs
}

// FillMissingSourceTags fills in missing source tags without overwriting existing values
func (s *Service) FillMissingSourceTags(runs []benchmark.Run, tag *config.TagConfig) []benchmark.Run {
	if tag == nil {
		return runs
	}

	filledCount := 0
	skippedCount := 0

	for i := range runs {
		if runs[i].TestConfig == nil {
			runs[i].TestConfig = make(map[string]interface{})
		}

		// Only fill if the tag doesn't exist or is empty
		if _, exists := runs[i].TestConfig[tag.Key]; !exists {
			runs[i].TestConfig[tag.Key] = tag.Value
			filledCount++
			s.log.Debug("Filled missing source tag", "runID", runs[i].ID, "key", tag.Key, "value", tag.Value)
		} else {
			skippedCount++
			s.log.Debug("Skipped existing source tag", "runID", runs[i].ID, "key", tag.Key, "existingValue", runs[i].TestConfig[tag.Key])
		}
	}

	s.log.Info("Filled missing source tags", "key", tag.Key, "value", tag.Value, "filled", filledCount, "skipped", skippedCount, "total", len(runs))
	return runs
}

// applyBenchmarkRunStrategy applies the BenchmarkRun strategy to imported runs
func (s *Service) applyBenchmarkRunStrategy(srcRuns []benchmark.Run, destMetadata *benchmark.RunGroup, strategy BenchmarkRunOption) ([]benchmark.Run, error) {
	var benchmarkRunID string

	switch strategy {
	case BenchmarkRunAddToLast:
		benchmarkRunID = s.getLastBenchmarkRunID(destMetadata)
		if benchmarkRunID == "" {
			// If no existing runs or no BenchmarkRun ID found, generate a new one
			var err error
			benchmarkRunID, err = s.generateRandomID()
			if err != nil {
				return nil, errors.Wrap(err, "failed to generate BenchmarkRun ID")
			}
			s.log.Info("No existing BenchmarkRun found, generated new ID", "benchmarkRunID", benchmarkRunID)
		} else {
			s.log.Info("Adding to existing BenchmarkRun", "benchmarkRunID", benchmarkRunID)
		}

	case BenchmarkRunCreateNew:
		var err error
		benchmarkRunID, err = s.generateRandomID()
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate new BenchmarkRun ID")
		}
		s.log.Info("Created new BenchmarkRun", "benchmarkRunID", benchmarkRunID)
	}

	// Apply BenchmarkRun ID to all imported runs
	for i := range srcRuns {
		if srcRuns[i].TestConfig == nil {
			srcRuns[i].TestConfig = make(map[string]interface{})
		}
		srcRuns[i].TestConfig[benchmark.BenchmarkRunTag] = benchmarkRunID
	}

	return srcRuns, nil
}

// MergeMetadata merges source metadata into destination metadata
func (s *Service) MergeMetadata(srcMetadata, destMetadata *benchmark.RunGroup, srcTag, destTag *config.TagConfig, benchmarkRunOpt BenchmarkRunOption) (*benchmark.RunGroup, *ImportSummary) {
	s.log.Info("Merging metadata", "src_runs", len(srcMetadata.Runs), "dest_runs", len(destMetadata.Runs))

	// Fill missing CreatedAt fields in source runs
	srcRuns := s.fillCreatedAt(srcMetadata.Runs, srcMetadata.CreatedAt)

	// Mark all imported runs as complete
	srcRuns = s.markRunsAsComplete(srcRuns)

	// Apply tags to source and destination runs
	srcRuns = s.ApplyTags(srcRuns, srcTag)                          // Apply destination tag to imported runs
	destRuns := s.FillMissingSourceTags(destMetadata.Runs, destTag) // Fill missing source tags without overwriting

	// Apply BenchmarkRun strategy to imported runs
	srcRuns, err := s.applyBenchmarkRunStrategy(srcRuns, destMetadata, benchmarkRunOpt)
	if err != nil {
		s.log.Error("Failed to apply BenchmarkRun strategy", "error", err)
		// Continue with import but log the error
	}

	// Check for conflicts (same run IDs)
	var conflicts []string
	existingIDs := make(map[string]bool)
	for _, run := range destRuns {
		existingIDs[run.ID] = true
	}

	for _, run := range srcRuns {
		if existingIDs[run.ID] {
			conflicts = append(conflicts, run.ID)
		}
	}

	// Merge runs (imported runs are added to the end)
	mergedRuns := append(destRuns, srcRuns...)

	summary := &ImportSummary{
		ImportedRunsCount: len(srcRuns),
		ExistingRunsCount: len(destRuns),
		SrcTagApplied:     srcTag,
		DestTagApplied:    destTag,
		Conflicts:         conflicts,
	}

	mergedMetadata := &benchmark.RunGroup{
		Runs: mergedRuns,
	}

	s.log.Info("Merged metadata", "total_runs", len(mergedRuns), "conflicts", len(conflicts))
	return mergedMetadata, summary
}

// WriteMetadata writes the merged metadata back to the output file
func (s *Service) WriteMetadata(metadata *benchmark.RunGroup) error {
	metadataPath := path.Join(s.config.OutputDir(), "metadata.json")
	s.log.Info("Writing merged metadata", "path", metadataPath, "runs", len(metadata.Runs))

	// Create backup of existing file
	backupPath := metadataPath + ".backup." + fmt.Sprintf("%d", time.Now().Unix())
	if _, err := os.Stat(metadataPath); err == nil {
		if err := os.Rename(metadataPath, backupPath); err != nil {
			s.log.Warn("Failed to create backup", "error", err)
		} else {
			s.log.Info("Created backup", "path", backupPath)
		}
	}

	// Write new metadata
	file, err := os.OpenFile(metadataPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to create metadata file")
	}
	defer func() { _ = file.Close() }()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		return errors.Wrap(err, "failed to encode metadata JSON")
	}

	s.log.Info("Successfully wrote merged metadata")
	return nil
}

// Import performs the complete import operation
func (s *Service) Import(request *ImportRequest) (*ImportResult, error) {
	s.log.Info("Starting import operation")

	// Merge metadata
	mergedMetadata, summary := s.MergeMetadata(
		request.SourceMetadata,
		request.DestMetadata,
		request.SrcTag,
		request.DestTag,
		request.BenchmarkRunOpt,
	)

	// Write merged metadata
	if err := s.WriteMetadata(mergedMetadata); err != nil {
		return &ImportResult{
			Success: false,
			Error:   err,
		}, err
	}

	result := &ImportResult{
		ImportedRuns: summary.ImportedRunsCount,
		UpdatedRuns:  summary.ExistingRunsCount,
		TotalRuns:    len(mergedMetadata.Runs),
		Success:      true,
	}

	s.log.Info("Import operation completed", "imported", result.ImportedRuns, "total", result.TotalRuns)
	return result, nil
}
