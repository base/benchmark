package benchmark

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// SnapshotDefinition is the user-facing YAML configuration for specifying
// a snapshot to be restored before running a benchmark.
type SnapshotDefinition struct {
	NodeType          string  `yaml:"node_type"`
	Command           string  `yaml:"command"`
	GenesisFile       *string `yaml:"genesis_file"`
	SuperchainChainID *uint64 `yaml:"superchain_chain_id"`
	ForceClean        *bool   `yaml:"force_clean"`
}

// CreateSnapshot copies the snapshot to the output directory for the given
// node type.
func (s SnapshotDefinition) CreateSnapshot(nodeType string, outputDir string) error {
	// default to true if not set
	forceClean := s.ForceClean == nil || *s.ForceClean
	if _, err := os.Stat(outputDir); err == nil && forceClean {
		// TODO: we could reuse it here potentially
		if err := os.RemoveAll(outputDir); err != nil {
			return fmt.Errorf("failed to remove existing snapshot: %w", err)
		}
	}

	// get absolute path of outputDir
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get absolute path of outputDir: %w", err)
	}

	outputDir = path.Join(currentDir, outputDir)

	var cmdBin string
	var args []string
	// split out default args from command
	parts := strings.SplitN(s.Command, " ", 2)
	if len(parts) < 2 {
		cmdBin = parts[0]
	} else {
		cmdBin = parts[0]
		args = strings.Split(parts[1], " ")
	}

	args = append(args, nodeType, outputDir)

	cmd := exec.Command(cmdBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// SnapshotManager is an interface that manages snapshots for different node types
// and roles.
type SnapshotManager interface {
	// EnsureInitialSnapshot ensures that an initial snapshot exists for the given node type.
	// If it does not exist, it will create it using the given snapshot definition.
	// Returns the path to the initial snapshot.
	EnsureInitialSnapshot(definition SnapshotDefinition) (string, error)

	// GetInitialSnapshotPath returns the path to the initial snapshot for the given node type.
	// Returns empty string if no initial snapshot exists for the node type.
	GetInitialSnapshotPath(nodeType string) string

	// CopyFromInitialSnapshot copies data from an initial snapshot to a test-specific directory.
	// This is used for per-test snapshots that need to be isolated from each other.
	CopyFromInitialSnapshot(initialSnapshotPath, testSnapshotPath string) error

	// EnsureSnapshot ensures that a snapshot exists for the given node type and
	// role. If it does not exist, it will create it using the given snapshot
	// definition. It returns the path to the snapshot.
	EnsureSnapshot(definition SnapshotDefinition, nodeType string, role string) (string, error)
}

type snapshotStoragePath struct {
	// nodeType is the type of node that is using this snapshot.
	nodeType string

	// role is "validator" or "sequencer". Each must have their own datadir
	// because we need to re-execute blocks from scratch on the validator.
	role string

	// command is the command that created this snapshot.
	command string
}

func (s *snapshotStoragePath) Equals(other *snapshotStoragePath) bool {
	if s.nodeType != other.nodeType {
		return false
	}
	if s.role != other.role {
		return false
	}
	if s.command != other.command {
		return false
	}
	return true
}

type benchmarkDatadirState struct {
	// currentDataDirs is a map of node types to their datadir. Datadirs can be
	// reused by multiple tests ro reduce the amount of copying that needs to be
	// done.
	currentDataDirs map[snapshotStoragePath]string

	// initialSnapshots tracks the paths to initial snapshots by node type
	initialSnapshots map[string]string

	// snapshotsDir is the directory where all the snapshots are stored. Each
	// file will have the format <nodeType>_<role>_<hash_command>.
	snapshotsDir string
}

func NewSnapshotManager(snapshotsDir string) SnapshotManager {
	return &benchmarkDatadirState{
		currentDataDirs:  make(map[snapshotStoragePath]string),
		initialSnapshots: make(map[string]string),
		snapshotsDir:     snapshotsDir,
	}
}

func (b *benchmarkDatadirState) EnsureInitialSnapshot(definition SnapshotDefinition) (string, error) {
	// Check if we already have this initial snapshot
	if path, exists := b.initialSnapshots[definition.NodeType]; exists {
		return path, nil
	}

	// Create the initial snapshot path
	hashCommand := sha256.New().Sum([]byte(definition.Command))
	initialSnapshotPath := filepath.Join(b.snapshotsDir, fmt.Sprintf("initial_%s_%x", definition.NodeType, hashCommand[:12]))

	// Create the initial snapshot
	err := definition.CreateSnapshot(definition.NodeType, initialSnapshotPath)
	if err != nil {
		return "", fmt.Errorf("failed to create initial snapshot: %w", err)
	}

	b.initialSnapshots[definition.NodeType] = initialSnapshotPath
	return initialSnapshotPath, nil
}

func (b *benchmarkDatadirState) GetInitialSnapshotPath(nodeType string) string {
	if path, exists := b.initialSnapshots[nodeType]; exists {
		return path
	}
	return ""
}

func (b *benchmarkDatadirState) CopyFromInitialSnapshot(initialSnapshotPath, testSnapshotPath string) error {
	// Remove existing test snapshot directory if it exists
	if _, err := os.Stat(testSnapshotPath); err == nil {
		if err := os.RemoveAll(testSnapshotPath); err != nil {
			return fmt.Errorf("failed to remove existing test snapshot: %w", err)
		}
	}

	// Create parent directory for test snapshot
	if err := os.MkdirAll(filepath.Dir(testSnapshotPath), 0755); err != nil {
		return fmt.Errorf("failed to create test snapshot parent directory: %w", err)
	}

	// Use rsync to copy the initial snapshot to the test location
	cmd := exec.Command("rsync", "-a", initialSnapshotPath+"/", testSnapshotPath+"/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy initial snapshot using rsync: %w", err)
	}

	return nil
}

func (b *benchmarkDatadirState) EnsureSnapshot(definition SnapshotDefinition, nodeType string, role string) (string, error) {
	snapshotDatadir := snapshotStoragePath{
		nodeType: nodeType,
		role:     role,
		command:  definition.Command,
	}

	if datadir, ok := b.currentDataDirs[snapshotDatadir]; ok {
		return datadir, nil
	}

	hashCommand := sha256.New().Sum([]byte(definition.Command))

	snapshotPath := filepath.Join(b.snapshotsDir, fmt.Sprintf("%s_%s_%x", nodeType, role, hashCommand[:12]))

	// Create a new datadir for this snapshot.
	err := definition.CreateSnapshot(nodeType, snapshotPath)
	if err != nil {
		return "", err
	}
	b.currentDataDirs[snapshotDatadir] = snapshotPath
	return snapshotPath, nil
}
