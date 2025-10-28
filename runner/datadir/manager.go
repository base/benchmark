package datadir

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/network/types"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// Manager handles the creation and management of test data directories
type Manager struct {
	// tracks persistent test directories for reuse_existing snapshots
	// key: nodeType, value: map["sequencer"|"validator"] -> TestDirConfig
	persistentTestDirs map[string]map[string]*TestDirConfig

	snapshotManager benchmark.SnapshotManager
	workingDir      string
	log             log.Logger
}

// TestDirConfig contains the configuration for a test directory
type TestDirConfig struct {
	SequencerOptions *config.InternalClientOptions
	ValidatorOptions *config.InternalClientOptions
}

// NewManager creates a new DataDirManager
func NewManager(workingDir string, snapshotManager benchmark.SnapshotManager, log log.Logger) *Manager {
	return &Manager{
		persistentTestDirs: make(map[string]map[string]*TestDirConfig),
		snapshotManager:    snapshotManager,
		workingDir:         workingDir,
		log:                log,
	}
}

// fileExists checks if a file exists
func (m *Manager) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SetupTestDirs sets up test directories for a benchmark run
// For reuse_existing snapshots, it creates persistent directories that will be reused across tests
// For other snapshot methods, directories will be created per-test in runTest
func (m *Manager) SetupTestDirs(params types.RunParams, genesis *core.Genesis, snapshot *benchmark.SnapshotDefinition, clientOptions config.ClientOptions) (*TestDirConfig, error) {
	isReuseExisting := snapshot != nil && snapshot.GetSnapshotMethod() == benchmark.SnapshotMethodReuseExisting

	if !isReuseExisting {
		return nil, nil
	}

	// For reuse_existing, create persistent directories
	nodeType := params.NodeType

	if _, exists := m.persistentTestDirs[nodeType]; !exists {
		m.persistentTestDirs[nodeType] = make(map[string]*TestDirConfig)
	}

	// Check if we already have persistent directories for this node type
	if existingConfig, exists := m.persistentTestDirs[nodeType]["config"]; exists {
		m.log.Info("Reusing existing persistent test directories", "nodeType", nodeType)
		return existingConfig, nil
	}

	// Create new persistent directories
	testName := fmt.Sprintf("persistent-%s", nodeType)
	sequencerTestDir := path.Join(m.workingDir, fmt.Sprintf("%s-sequencer", testName))
	validatorTestDir := path.Join(m.workingDir, fmt.Sprintf("%s-validator", testName))

	m.log.Info("Creating persistent test directories for reuse_existing",
		"nodeType", nodeType,
		"sequencer", sequencerTestDir,
		"validator", validatorTestDir)

	// Setup data directories
	sequencerOptions, validatorOptions, err := m.setupDataDirs(sequencerTestDir, validatorTestDir, params, genesis, snapshot, clientOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup data dirs")
	}

	testDirConfig := &TestDirConfig{
		SequencerOptions: sequencerOptions,
		ValidatorOptions: validatorOptions,
	}

	m.persistentTestDirs[nodeType]["config"] = testDirConfig

	return testDirConfig, nil
}

// GetOrCreateTestDirs gets existing persistent directories or creates temporary ones
func (m *Manager) GetOrCreateTestDirs(params types.RunParams, genesis *core.Genesis, snapshot *benchmark.SnapshotDefinition, clientOptions config.ClientOptions, testTimestamp int64) (*config.InternalClientOptions, *config.InternalClientOptions, bool, error) {
	isReuseExisting := snapshot != nil && snapshot.GetSnapshotMethod() == benchmark.SnapshotMethodReuseExisting

	if isReuseExisting {
		// Return pre-configured persistent directories
		if config, exists := m.persistentTestDirs[params.NodeType]["config"]; exists {
			m.log.Info("Using persistent test directories", "nodeType", params.NodeType)
			return config.SequencerOptions, config.ValidatorOptions, false, nil // false = don't cleanup
		}
		return nil, nil, false, fmt.Errorf("persistent directories not setup for node type %s", params.NodeType)
	}

	// For non-reuse_existing, create temporary directories
	testName := fmt.Sprintf("%d-%s-test", testTimestamp, params.NodeType)
	sequencerTestDir := path.Join(m.workingDir, fmt.Sprintf("%s-sequencer", testName))
	validatorTestDir := path.Join(m.workingDir, fmt.Sprintf("%s-validator", testName))

	sequencerOptions, validatorOptions, err := m.setupDataDirs(sequencerTestDir, validatorTestDir, params, genesis, snapshot, clientOptions)
	if err != nil {
		return nil, nil, false, errors.Wrap(err, "failed to setup data dirs")
	}

	return sequencerOptions, validatorOptions, true, nil // true = cleanup after test
}

// setupDataDirs sets up the data directories for sequencer and validator
func (m *Manager) setupDataDirs(sequencerTestDir string, validatorTestDir string, params types.RunParams, genesis *core.Genesis, snapshot *benchmark.SnapshotDefinition, clientOptions config.ClientOptions) (*config.InternalClientOptions, *config.InternalClientOptions, error) {
	var sequencerDataDirOverride, validatorDataDirOverride string

	if snapshot != nil && snapshot.GetSnapshotMethod() == benchmark.SnapshotMethodReuseExisting {
		sequencerDataDirOverride = path.Join(sequencerTestDir, "data")
		validatorDataDirOverride = path.Join(validatorTestDir, "data")

		// Check if this is the first run (directories don't exist yet)
		isFirstRun := !m.fileExists(sequencerDataDirOverride) && !m.fileExists(validatorDataDirOverride)

		if isFirstRun {
			initialSnapshotPath := m.snapshotManager.GetInitialSnapshotPath(params.NodeType)
			if initialSnapshotPath != "" && m.fileExists(initialSnapshotPath) {
				m.log.Info("First run with reuse_existing: copying to validator, moving to sequencer",
					"initialSnapshot", initialSnapshotPath,
					"sequencerDataDir", sequencerDataDirOverride,
					"validatorDataDir", validatorDataDirOverride)

				// First: copy from initial snapshot to validator directory
				err := m.snapshotManager.CopyFromInitialSnapshot(initialSnapshotPath, validatorDataDirOverride)
				if err != nil {
					return nil, nil, errors.Wrap(err, "failed to copy initial snapshot to validator directory")
				}
				m.log.Info("Copied initial snapshot to validator directory", "path", validatorDataDirOverride)

				err = os.MkdirAll(sequencerTestDir, 0755)
				if err != nil {
					return nil, nil, errors.Wrap(err, "failed to create sequencer test directory")
				}

				err = os.Rename(initialSnapshotPath, sequencerDataDirOverride)
				if err != nil {
					return nil, nil, errors.Wrap(err, "failed to move initial snapshot to sequencer directory")
				}
				m.log.Info("Moved initial snapshot to sequencer directory", "from", initialSnapshotPath, "to", sequencerDataDirOverride)
			}
		} else {
			m.log.Info("Reusing existing data directories from previous run",
				"sequencerDataDir", sequencerDataDirOverride,
				"validatorDataDir", validatorDataDirOverride)
		}
	}

	sequencerOptions, err := m.setupInternalDirectories(sequencerTestDir, params, genesis, snapshot, "sequencer", sequencerDataDirOverride, clientOptions)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to setup internal directories")
	}

	validatorOptions, err := m.setupInternalDirectories(validatorTestDir, params, genesis, snapshot, "validator", validatorDataDirOverride, clientOptions)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to setup internal directories")
	}

	return sequencerOptions, validatorOptions, nil
}

// setupInternalDirectories sets up the internal directory structure for a test
func (m *Manager) setupInternalDirectories(testDir string, params types.RunParams, genesis *core.Genesis, snapshot *benchmark.SnapshotDefinition, role string, dataDirOverride string, clientOptions config.ClientOptions) (*config.InternalClientOptions, error) {
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create test directory")
	}

	metricsPath := path.Join(testDir, "metrics")
	// Use MkdirAll to avoid error if directory already exists
	err = os.MkdirAll(metricsPath, 0755)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create metrics directory")
	}

	// write chain config to testDir/chain.json
	chainCfgPath := path.Join(testDir, "chain.json")
	// Only create chain config if it doesn't exist (for reuse_existing)
	if !m.fileExists(chainCfgPath) {
		chainCfgFile, err := os.OpenFile(chainCfgPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open chain config file")
		}

		err = json.NewEncoder(chainCfgFile).Encode(genesis)
		if err != nil {
			return nil, errors.Wrap(err, "failed to write chain config")
		}
		if err := chainCfgFile.Close(); err != nil {
			return nil, errors.Wrap(err, "failed to close chain config file")
		}
	}

	var dataDirPath string
	var isSnapshot bool

	// If dataDirOverride is provided, use it (already set up by caller)
	if dataDirOverride != "" {
		dataDirPath = dataDirOverride
		isSnapshot = true // dataDirOverride is only set when using snapshots
		m.log.Info("Using pre-configured datadir", "path", dataDirPath, "role", role)
	} else {
		isSnapshot = snapshot != nil && snapshot.Command != ""
		if isSnapshot {
			dataDirPath = path.Join(testDir, "data")

			initialSnapshotPath := m.snapshotManager.GetInitialSnapshotPath(params.NodeType)

			if initialSnapshotPath != "" && m.fileExists(initialSnapshotPath) {
				snapshotMethod := snapshot.GetSnapshotMethod()

				switch snapshotMethod {
				case benchmark.SnapshotMethodReuseExisting:
					dataDirPath = initialSnapshotPath
					m.log.Info("Reusing existing snapshot", "snapshotPath", initialSnapshotPath, "method", snapshotMethod)
				case benchmark.SnapshotMethodHeadRollback:
					// For head_rollback, copy the snapshot but mark it for rollback later
					err := m.snapshotManager.CopyFromInitialSnapshot(initialSnapshotPath, dataDirPath)
					if err != nil {
						return nil, errors.Wrap(err, "failed to copy from initial snapshot for head rollback")
					}
					m.log.Info("Copied from initial snapshot for head rollback", "initialSnapshotPath", initialSnapshotPath, "dataDirPath", dataDirPath, "method", snapshotMethod)
				default:
					// Default chain_copy behavior
					err := m.snapshotManager.CopyFromInitialSnapshot(initialSnapshotPath, dataDirPath)
					if err != nil {
						return nil, errors.Wrap(err, "failed to copy from initial snapshot")
					}
					m.log.Info("Copied from initial snapshot", "initialSnapshotPath", initialSnapshotPath, "dataDirPath", dataDirPath)
				}
			} else {
				// Fallback to direct snapshot creation
				if initialSnapshotPath != "" {
					m.log.Warn("Initial snapshot path registered but doesn't exist, falling back to direct snapshot creation",
						"path", initialSnapshotPath, "nodeType", params.NodeType)
				}
				snapshotDir, err := m.snapshotManager.EnsureSnapshot(*snapshot, params.NodeType, role)
				if err != nil {
					return nil, errors.Wrap(err, "failed to ensure snapshot")
				}
				dataDirPath = snapshotDir
			}
		} else {
			// if no snapshot, just create a new datadir
			dataDirPath = path.Join(testDir, "data")
			err = os.Mkdir(dataDirPath, 0755)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create data directory")
			}
		}
	}

	jwtSecretPath := path.Join(testDir, "jwt_secret")
	var jwtSecretStr string

	// Check if JWT secret already exists (for reuse_existing)
	if m.fileExists(jwtSecretPath) {
		jwtSecretBytes, err := os.ReadFile(jwtSecretPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read existing jwt secret")
		}
		jwtSecretStr = string(jwtSecretBytes)
		m.log.Info("Reusing existing JWT secret", "path", jwtSecretPath, "role", role)
	} else {
		// Generate new JWT secret
		var jwtSecret [32]byte
		_, err = rand.Read(jwtSecret[:])
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate jwt secret")
		}

		jwtSecretFile, err := os.OpenFile(jwtSecretPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open jwt secret file")
		}

		jwtSecretStr = hex.EncodeToString(jwtSecret[:])
		_, err = jwtSecretFile.Write([]byte(jwtSecretStr))
		if err != nil {
			return nil, errors.Wrap(err, "failed to write jwt secret")
		}

		if err = jwtSecretFile.Close(); err != nil {
			return nil, errors.Wrap(err, "failed to close jwt secret file")
		}
		m.log.Info("Generated new JWT secret", "path", jwtSecretPath, "role", role)
	}

	options := clientOptions
	options = params.ClientOptions(options)

	options.SkipInit = isSnapshot

	internalOptions := &config.InternalClientOptions{
		ClientOptions: options,
		JWTSecretPath: jwtSecretPath,
		MetricsPath:   metricsPath,
		JWTSecret:     jwtSecretStr,
		ChainCfgPath:  chainCfgPath,
		DataDirPath:   dataDirPath,
		TestDirPath:   testDir,
	}

	return internalOptions, nil
}
