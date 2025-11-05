package runner

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/benchmark/portmanager"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/datadir"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network"
	"github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload"
	"github.com/base/base-bench/runner/utils"
	"github.com/ethereum/go-ethereum/core"
	ethparams "github.com/ethereum/go-ethereum/params"
)

var ErrAlreadyStopped = errors.New("already stopped")

type Service interface {
	Run(ctx context.Context) error
}

type service struct {
	// tracks the state of the datadirs for each test
	// this is used to avoid copying the datadirs for each test
	dataDirState   benchmark.SnapshotManager
	dataDirManager *datadir.Manager
	portState      portmanager.PortManager
	metadataPath   string

	config  config.Config
	version string
	log     log.Logger
}

func NewService(version string, cfg config.Config, log log.Logger) Service {
	metadataPath := path.Join(cfg.OutputDir(), "metadata.json")
	snapshotManager := benchmark.NewSnapshotManager(path.Join(cfg.DataDir(), "snapshots"))

	s := &service{
		metadataPath:   metadataPath,
		portState:      portmanager.NewPortManager(),
		dataDirState:   snapshotManager,
		dataDirManager: datadir.NewManager(cfg.DataDir(), snapshotManager, log),
		config:         cfg,
		version:        version,
		log:            log,
	}

	return s
}

func readBenchmarkConfig(path string) (*benchmark.BenchmarkConfig, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}

	var config *benchmark.BenchmarkConfig
	err = yaml.NewDecoder(file).Decode(&config)
	return config, err
}

type TestRunMetadata struct {
	TestName string  `json:"test_name"`
	Success  bool    `json:"success"`
	Error    *string `json:"error,omitempty"`
}

func (s *service) exportOutput(testName string, returnedError error, testDirs *config.InternalClientOptions, testOutputDir string, nodeType string) error {
	// package up logs from the EL client and write them to the output dir
	// copy metrics.json to output dir
	metricsPath := path.Join(testDirs.MetricsPath, metrics.MetricsFileName)
	metricsOutputPath := path.Join(testOutputDir, fmt.Sprintf("metrics-%s.json", nodeType))
	err := os.Rename(metricsPath, metricsOutputPath)
	if err != nil {
		return errors.Wrap(err, "failed to move metrics file")
	}

	// copy logs to output dir gzipped
	logsPath := path.Join(testDirs.TestDirPath, network.ExecutionLayerLogFileName)
	logsOutputPath := path.Join(testOutputDir, fmt.Sprintf("logs-%s.gz", nodeType))

	outFile, err := os.OpenFile(logsOutputPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open logs file")
	}
	defer func() {
		_ = outFile.Close()
	}()

	wr := gzip.NewWriter(outFile)
	defer func() {
		_ = wr.Close()
	}()

	inFile, err := os.Open(logsPath)
	if err != nil {
		return errors.Wrap(err, "failed to open logs file")
	}
	defer func() {
		_ = inFile.Close()
	}()
	_, err = io.Copy(wr, inFile)
	if err != nil {
		return errors.Wrap(err, "failed to copy logs file")
	}
	_ = wr.Close()

	logsFile, err := os.Open(logsPath)
	if err != nil {
		return errors.Wrap(err, "failed to open logs file")
	}
	defer func() {
		_ = logsFile.Close()
	}()

	errStr := (*string)(nil)
	if returnedError != nil {
		errStr = new(string)
		*errStr = returnedError.Error()
	}

	// write result.json
	metadata := TestRunMetadata{
		TestName: testName,
		Success:  errStr == nil,
		Error:    errStr,
	}

	resultPath := path.Join(testOutputDir, fmt.Sprintf("result-%s.json", nodeType))
	resultFile, err := os.OpenFile(resultPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open result file")
	}

	jsonEncoder := json.NewEncoder(resultFile)
	jsonEncoder.SetIndent("", "  ")
	err = jsonEncoder.Encode(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to write result file")
	}
	if err = resultFile.Close(); err != nil {
		return errors.Wrap(err, "failed to close result file")
	}

	return nil
}

func (s *service) getGenesisForSnapshotConfig(snapshotConfig *benchmark.SnapshotDefinition) (*core.Genesis, error) {
	usingSnapshot := snapshotConfig != nil && snapshotConfig.Command != ""
	var genesis *core.Genesis

	if usingSnapshot {
		if snapshotConfig.GenesisFile != nil {
			s.log.Info("Using genesis file from snapshot config", "command", snapshotConfig.Command, "genesis_file", *snapshotConfig.GenesisFile)

			// read genesis file
			genesisFile, err := os.Open(*snapshotConfig.GenesisFile)
			if err != nil {
				return nil, errors.Wrap(err, "failed to open genesis file")
			}

			defer func() {
				_ = genesisFile.Close()
			}()

			genesis = new(core.Genesis)
			err = json.NewDecoder(genesisFile).Decode(genesis)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode genesis file")
			}
		} else {
			s.log.Info("Using genesis file from superchain", "command", snapshotConfig.Command, "superchain_chain_id", *snapshotConfig.SuperchainChainID)

			genesis, err := core.LoadOPStackGenesis(*snapshotConfig.SuperchainChainID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to load genesis file")
			}

			return genesis, nil
		}
	} else {
		// for devnets, just create a new genesis with the current time
		genesis = benchmark.DefaultDevnetGenesis()
	}

	return genesis, nil
}

func (s *service) setupBlobsDir(workingDir string) error {
	// create temp directory for blobs
	blobsDir := path.Join(workingDir, "blobs")
	err := os.MkdirAll(blobsDir, 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create blobs directory")
	}
	return nil
}

func (s *service) setupInitialSnapshots(testPlans []benchmark.TestPlan) error {
	// Collect all unique initial snapshots across all test plans
	initialSnapshotsMap := make(map[string]benchmark.SnapshotDefinition)

	for _, testPlan := range testPlans {
		for _, snapshot := range testPlan.InitialSnapshots {
			// Use node_type as the key to avoid duplicates
			initialSnapshotsMap[snapshot.NodeType] = snapshot
		}
	}

	// Setup each unique initial snapshot
	for nodeType, snapshot := range initialSnapshotsMap {
		s.log.Info("Setting up initial snapshot from remote source", "node_type", nodeType, "command", snapshot.Command)
		_, err := s.dataDirState.EnsureInitialSnapshot(snapshot)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to setup initial snapshot for node type %s", nodeType))
		}
		s.log.Info("Initial snapshot setup completed", "node_type", nodeType)
	}

	return nil
}

func (s *service) setupPersistentTestDirs(testPlans []benchmark.TestPlan) error {
	// Collect all test runs that use reuse_existing snapshots
	testRunsWithReuseExisting := make(map[string]struct {
		params   types.RunParams
		genesis  *core.Genesis
		snapshot *benchmark.SnapshotDefinition
	})

	for _, testPlan := range testPlans {
		for _, run := range testPlan.Runs {
			// Find the appropriate snapshot for this node type
			var snapshotConfig *benchmark.SnapshotDefinition
			for _, s := range testPlan.InitialSnapshots {
				if s.NodeType == run.Params.NodeType {
					snapshotConfig = &s
					break
				}
			}

			// Only setup if using reuse_existing method
			if snapshotConfig != nil && snapshotConfig.GetSnapshotMethod() == benchmark.SnapshotMethodReuseExisting {
				// Get genesis for this snapshot
				genesis, err := s.getGenesisForSnapshotConfig(snapshotConfig)
				if err != nil {
					return errors.Wrap(err, "failed to get genesis block")
				}

				testRunsWithReuseExisting[run.Params.NodeType] = struct {
					params   types.RunParams
					genesis  *core.Genesis
					snapshot *benchmark.SnapshotDefinition
				}{
					params:   run.Params,
					genesis:  genesis,
					snapshot: snapshotConfig,
				}
			}
		}
	}

	// Setup persistent directories for each unique node type with reuse_existing
	// This will create separate snapshots for sequencer and validator roles
	for nodeType, runInfo := range testRunsWithReuseExisting {
		s.log.Info("Setting up persistent test directories for both roles", "nodeType", nodeType)
		err := s.dataDirManager.SetupTestDirs(runInfo.params, runInfo.genesis, runInfo.snapshot, s.config.ClientOptions())
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to setup persistent test directories for node type %s", nodeType))
		}
		s.log.Info("Persistent test directories setup completed for both sequencer and validator", "nodeType", nodeType)
	}

	return nil
}

func (s *service) detectSnapshotHeadBlocks(testPlans []benchmark.TestPlan) error {
	// Collect all unique snapshots that use head_rollback method
	snapshotsToDetect := make(map[string]benchmark.SnapshotDefinition)

	for _, testPlan := range testPlans {
		for _, snapshot := range testPlan.InitialSnapshots {
			if snapshot.GetSnapshotMethod() == benchmark.SnapshotMethodHeadRollback && snapshot.RollbackBlock == nil {
				// Only detect head block if rollback_block is not specified
				snapshotsToDetect[snapshot.NodeType] = snapshot
			}
		}
	}

	if len(snapshotsToDetect) == 0 {
		return nil // No head block detection needed
	}

	s.log.Info("Detecting head blocks for snapshots", "count", len(snapshotsToDetect))

	// Detect head block for each snapshot that needs it
	for nodeType, snapshot := range snapshotsToDetect {
		headBlock, err := s.detectHeadBlockForSnapshot(nodeType, snapshot)
		if err != nil {
			s.log.Warn("Failed to detect head block for snapshot", "nodeType", nodeType, "error", err)
			continue
		}

		s.log.Info("Detected head block for snapshot", "nodeType", nodeType, "headBlock", headBlock)
		// Store the detected head block back into the snapshot manager
		// We'll need to update the test plans to use this detected block
		s.updateTestPlansWithDetectedHeadBlock(testPlans, nodeType, headBlock)
	}

	return nil
}

func (s *service) detectHeadBlockForSnapshot(nodeType string, snapshot benchmark.SnapshotDefinition) (uint64, error) {
	// Get the snapshot path
	snapshotPath := s.dataDirState.GetInitialSnapshotPath(nodeType)
	if snapshotPath == "" {
		return 0, fmt.Errorf("no initial snapshot path found for node type %s", nodeType)
	}

	// Create a temporary working directory for head block detection
	tempDir, err := os.MkdirTemp(s.config.DataDir(), fmt.Sprintf("head-detect-%s-*", nodeType))
	if err != nil {
		return 0, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			s.log.Warn("Failed to remove temp directory", "path", tempDir, "error", err)
		}
	}()

	// Copy the snapshot to the temp directory
	tempSnapshotPath := path.Join(tempDir, "data")
	err = s.dataDirState.CopyFromInitialSnapshot(snapshotPath, tempSnapshotPath)
	if err != nil {
		return 0, fmt.Errorf("failed to copy snapshot for head detection: %w", err)
	}

	// Create a temporary client configuration to detect the head block
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Setup temporary client options
	clientOptions := s.config.ClientOptions()
	// Set SkipInit for geth since we're using a snapshot
	clientOptions.SkipInit = true

	tempOptions := &config.InternalClientOptions{
		ClientOptions: clientOptions,
		DataDirPath:   tempSnapshotPath,
		TestDirPath:   tempDir,
	}

	// Set the appropriate binary path based on node type
	switch nodeType {
	case "geth":
		tempOptions.GethBin = s.config.ClientOptions().GethBin
	case "reth":
		tempOptions.RethBin = s.config.ClientOptions().RethBin
	case "rbuilder":
		tempOptions.RbuilderBin = s.config.ClientOptions().RbuilderBin
	}

	// Create JWT secret for the temporary client
	var jwtSecret [32]byte
	_, err = rand.Read(jwtSecret[:])
	if err != nil {
		return 0, fmt.Errorf("failed to generate jwt secret: %w", err)
	}

	jwtSecretPath := path.Join(tempDir, "jwt_secret")
	jwtSecretFile, err := os.OpenFile(jwtSecretPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to create jwt secret file: %w", err)
	}

	_, err = jwtSecretFile.Write([]byte(hex.EncodeToString(jwtSecret[:])))
	if err != nil {
		if closeErr := jwtSecretFile.Close(); closeErr != nil {
			s.log.Warn("Failed to close jwt secret file after write error", "error", closeErr)
		}
		return 0, fmt.Errorf("failed to write jwt secret: %w", err)
	}
	if err := jwtSecretFile.Close(); err != nil {
		return 0, fmt.Errorf("failed to close jwt secret file: %w", err)
	}

	tempOptions.JWTSecretPath = jwtSecretPath
	tempOptions.JWTSecret = hex.EncodeToString(jwtSecret[:])

	// Create a minimal genesis for the temp client
	genesis := benchmark.DefaultDevnetGenesis()
	chainCfgPath := path.Join(tempDir, "chain.json")
	chainCfgFile, err := os.OpenFile(chainCfgPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to create chain config file: %w", err)
	}

	err = json.NewEncoder(chainCfgFile).Encode(genesis)
	if closeErr := chainCfgFile.Close(); closeErr != nil {
		s.log.Warn("Failed to close chain config file", "error", closeErr)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to write chain config: %w", err)
	}

	tempOptions.ChainCfgPath = chainCfgPath

	// Start the temporary client to query head block
	tempClient, _, err := network.SetupNodeForHeadDetection(ctx, s.log, nodeType, tempOptions, s.portState)
	if err != nil {
		return 0, fmt.Errorf("failed to setup temporary client for head detection: %w", err)
	}
	defer tempClient.Stop()

	// Query the head block
	currentHeader, err := tempClient.Client().HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to query head block: %w", err)
	}

	return currentHeader.Number.Uint64(), nil
}

func (s *service) updateTestPlansWithDetectedHeadBlock(testPlans []benchmark.TestPlan, nodeType string, headBlock uint64) {
	for i := range testPlans {
		for j := range testPlans[i].InitialSnapshots {
			snapshot := &testPlans[i].InitialSnapshots[j]
			if snapshot.NodeType == nodeType && snapshot.GetSnapshotMethod() == benchmark.SnapshotMethodHeadRollback && snapshot.RollbackBlock == nil {
				snapshot.RollbackBlock = &headBlock
				s.log.Info("Updated snapshot with detected head block", "nodeType", nodeType, "headBlock", headBlock)
			}
		}
	}
}

func (s *service) runTest(ctx context.Context, params types.RunParams, workingDir string, outputDir string, snapshots []benchmark.SnapshotDefinition, proofConfig *benchmark.ProofProgramOptions, transactionPayload payload.Definition) (*benchmark.RunResult, error) {

	s.log.Info(fmt.Sprintf("Running benchmark with params: %+v", params))

	// Find the appropriate snapshot for this node type
	var snapshotConfig *benchmark.SnapshotDefinition
	for _, s := range snapshots {
		if s.NodeType == params.NodeType {
			snapshotConfig = &s
			break
		}
	}

	// get genesis block
	genesis, err := s.getGenesisForSnapshotConfig(snapshotConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get genesis block")
	}

	// Get or create test directories using the datadir manager
	testTimestamp := time.Now().Unix()
	sequencerOptions, validatorOptions, shouldCleanup, err := s.dataDirManager.GetOrCreateTestDirs(params, genesis, snapshotConfig, s.config.ClientOptions(), testTimestamp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get or create test dirs")
	}

	if proofConfig != nil {
		if err := s.setupBlobsDir(workingDir); err != nil {
			return nil, errors.Wrap(err, "failed to setup blobs directory")
		}
	}

	// Only cleanup directories if shouldCleanup is true (non-reuse_existing)
	if shouldCleanup {
		defer func() {
			// clean up test directory
			err := os.RemoveAll(sequencerOptions.TestDirPath)
			if err != nil {
				log.Error("failed to remove sequencer test directory", "err", err)
			}

			// clean up test directory
			err = os.RemoveAll(validatorOptions.TestDirPath)
			if err != nil {
				log.Error("failed to remove validator test directory", "err", err)
			}
		}()
	}

	// Generate testName for output
	testName := fmt.Sprintf("%d-%s-test", testTimestamp, params.NodeType)

	batcherKeyBytes := common.FromHex("0xd2ba8e70072983384203c438d4e94bf399cbd88bbcafb82b61cc96ed12541707")
	batcherKey, err := crypto.ToECDSA(batcherKeyBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create batcher key")
	}

	prefundKeyBytes := common.FromHex("0xad0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	prefundKey, err := crypto.ToECDSA(prefundKeyBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create prefund key")
	}

	prefundAmount := new(big.Int).Mul(big.NewInt(1e6), big.NewInt(ethparams.Ether))
	config := &types.TestConfig{
		Params:            params,
		Config:            s.config,
		Genesis:           *genesis,
		BatcherKey:        *batcherKey,
		PrefundPrivateKey: *prefundKey,
		PrefundAmount:     *prefundAmount,
	}

	// Run benchmark
	benchmark, err := network.NewNetworkBenchmark(config, s.log, sequencerOptions, validatorOptions, proofConfig, transactionPayload, s.portState)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create network benchmark")
	}

	// Set snapshot config for head rollback if needed
	if snapshotConfig != nil {
		benchmark.SetSnapshotConfig(*snapshotConfig)
	}
	err = benchmark.Run(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run benchmark")
	}

	err = s.exportOutput(testName, err, sequencerOptions, outputDir, "sequencer")
	if err != nil {
		return nil, errors.Wrap(err, "failed to export sequencer output")
	}

	err = s.exportOutput(testName, err, validatorOptions, outputDir, "validator")
	if err != nil {
		return nil, errors.Wrap(err, "failed to export validator output")
	}

	result, err := benchmark.GetResult()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get metrics")
	}

	return result, nil
}

func (s *service) readTestMetadata() ([]benchmark.Run, error) {
	existingMetadataFile, err := os.ReadFile(s.metadataPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read existing test metadata")
	}

	var existingMetadata benchmark.RunGroup
	err = json.Unmarshal(existingMetadataFile, &existingMetadata)
	if err != nil {
		return nil, err
	}

	return existingMetadata.Runs, nil
}

func (s *service) writeTestMetadata(testPlan benchmark.RunGroup) error {
	var runs []benchmark.Run

	existingRuns, _ := s.readTestMetadata()

	metadataFile, err := os.OpenFile(s.metadataPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open test metadata file")
	}

	// remove all runs that have the same benchmark run id
	for _, run := range existingRuns {
		if run.TestConfig["BenchmarkRun"] != testPlan.Runs[0].TestConfig["BenchmarkRun"] {
			runs = append(runs, run)
		}
	}

	runs = append(runs, testPlan.Runs...)

	newMetadata := benchmark.RunGroup{
		Runs: runs,
	}

	jsonEncoder := json.NewEncoder(metadataFile)
	jsonEncoder.SetIndent("", "  ")
	err = jsonEncoder.Encode(newMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to encode test metadata")
	}

	return nil
}

func (s *service) Run(ctx context.Context) error {
	s.log.Info("Starting")

	config, err := readBenchmarkConfig(s.config.ConfigPath())
	if err != nil {
		return errors.Wrap(err, "failed to read benchmark config")
	}

	numSuccess := 0
	numFailure := 0

	var testPlans []benchmark.TestPlan

	for _, c := range config.Benchmarks {
		testPlan, err := benchmark.NewTestPlanFromConfig(c, s.config.ConfigPath(), config)
		if err != nil {
			return errors.Wrap(err, "failed to create params matrix")
		}

		// add all the params to the test plan
		testPlans = append(testPlans, *testPlan)
	}

	// ensure output directory exists
	err = os.MkdirAll(s.config.OutputDir(), 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create output directory")
	}

	// Generate or use provided BenchmarkRunID for this session
	var benchmarkRunID string
	if s.config.BenchmarkRunID() != "" {
		benchmarkRunID = s.config.BenchmarkRunID()
		s.log.Info("Using provided BenchmarkRunID", "benchmarkRunID", benchmarkRunID)
	} else {
		// Use the first test run's ID as the benchmark run ID for this session
		if len(testPlans) > 0 && len(testPlans[0].Runs) > 0 {
			benchmarkRunID = testPlans[0].Runs[0].ID
			s.log.Info("Using first test run ID as BenchmarkRunID", "benchmarkRunID", benchmarkRunID)
		} else {
			// Fallback to generating one if no test runs exist
			benchmarkRunID, err = utils.GenerateRandomID(8)
			if err != nil {
				return errors.Wrap(err, "failed to generate BenchmarkRunID")
			}
			s.log.Info("Generated fallback BenchmarkRunID", "benchmarkRunID", benchmarkRunID)
		}
	}

	// Setup initial snapshots for all test plans before running any tests
	err = s.setupInitialSnapshots(testPlans)
	if err != nil {
		return errors.Wrap(err, "failed to setup initial snapshots")
	}

	// Detect head blocks for snapshots that use head_rollback method
	err = s.detectSnapshotHeadBlocks(testPlans)
	if err != nil {
		return errors.Wrap(err, "failed to detect snapshot head blocks")
	}

	// Setup persistent test directories for reuse_existing snapshots before running any tests
	err = s.setupPersistentTestDirs(testPlans)
	if err != nil {
		return errors.Wrap(err, "failed to setup persistent test directories")
	}

	// Create machine info from config
	var machineInfo *benchmark.MachineInfo
	s.log.Info("Machine info config values",
		"type", s.config.MachineType(),
		"provider", s.config.MachineProvider(),
		"region", s.config.MachineRegion(),
		"fileSystem", s.config.FileSystem())

	if s.config.MachineType() != "" || s.config.MachineProvider() != "" || s.config.MachineRegion() != "" || s.config.FileSystem() != "" {
		machineInfo = &benchmark.MachineInfo{
			Type:       s.config.MachineType(),
			Provider:   s.config.MachineProvider(),
			Region:     s.config.MachineRegion(),
			FileSystem: s.config.FileSystem(),
		}
		s.log.Info("Created machine info", "machineInfo", machineInfo)
	} else {
		s.log.Warn("No machine info available - all config values are empty")
	}

	metadata := benchmark.RunGroupFromTestPlans(testPlans, machineInfo)

	// Apply BenchmarkRunID to all runs in metadata
	for i := range metadata.Runs {
		if metadata.Runs[i].TestConfig == nil {
			metadata.Runs[i].TestConfig = make(map[string]interface{})
		}
		metadata.Runs[i].TestConfig[benchmark.BenchmarkRunTag] = benchmarkRunID
	}

	runIdx := 0

	// create map of transaction payloads
	transactionPayloads := make(map[string]payload.Definition)
	for _, w := range config.TransactionPayloads {
		if _, ok := transactionPayloads[w.ID]; ok {
			return fmt.Errorf("duplicate transaction payloads %s exist", w.ID)
		}

		transactionPayloads[w.ID] = w
	}

outerLoop:
	for _, testPlan := range testPlans {
		err = s.writeTestMetadata(metadata)
		if err != nil {
			return errors.Wrap(err, "failed to write test metadata")
		}

		for _, c := range testPlan.Runs {
			outputDir := path.Join(s.config.OutputDir(), c.OutputDir)

			// ensure output directory exists
			err = os.MkdirAll(outputDir, 0755)
			if err != nil {
				return errors.Wrap(err, "failed to create output directory")
			}

			metricSummary, err := s.runTest(ctx, c.Params, s.config.DataDir(), outputDir, testPlan.InitialSnapshots, testPlan.ProofProgram, transactionPayloads[c.Params.PayloadID])
			if err != nil {
				log.Error("Failed to run test", "err", err)
				metricSummary = &benchmark.RunResult{
					Success:  false,
					Complete: true,
				}
				numFailure++
			} else {
				numSuccess++
			}
			metadata.AddResult(runIdx, *metricSummary)

			err = s.writeTestMetadata(metadata)
			if err != nil {
				return errors.Wrap(err, "failed to write test metadata")
			}

			runIdx++

			select {
			case <-ctx.Done():
				// if ctx is done, stop running tests immediately
				break outerLoop
			default:
				continue
			}
		}
	}

	// after benchmarking, set all test runs to complete and write metadata
	for i := range metadata.Runs {
		if metadata.Runs[i].Result == nil {
			metadata.Runs[i].Result = &benchmark.RunResult{
				Success:  false,
				Complete: true,
			}
		} else if !metadata.Runs[i].Result.Complete {
			metadata.Runs[i].Result.Complete = true
		}
	}
	err = s.writeTestMetadata(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to write test metadata")
	}

	s.log.Info("Finished benchmarking", "numSuccess", numSuccess, "numFailure", numFailure)

	if numFailure > 0 {
		return fmt.Errorf("failed to run %d tests", numFailure)
	}

	return nil
}
