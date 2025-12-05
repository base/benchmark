package benchmark

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
)

// SnapshotManager is an interface that manages snapshots for different node types
// and roles.
type SnapshotManager interface {
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

	// snapshotsDir is the directory where all the snapshots are stored. Each
	// file will have the format <nodeType>_<role>_<hash_command>.
	snapshotsDir string
}

func NewSnapshotManager(snapshotsDir string) SnapshotManager {
	return &benchmarkDatadirState{
		currentDataDirs: make(map[snapshotStoragePath]string),
		snapshotsDir:    snapshotsDir,
	}
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
