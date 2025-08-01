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
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network"
	"github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload"
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
	dataDirState benchmark.SnapshotManager
	metadataPath string

	config  config.Config
	version string
	log     log.Logger
}

func NewService(version string, cfg config.Config, log log.Logger) Service {
	metadataPath := path.Join(cfg.OutputDir(), "metadata.json")

	return &service{
		metadataPath: metadataPath,
		dataDirState: benchmark.NewSnapshotManager(path.Join(cfg.DataDir(), "snapshots")),
		config:       cfg,
		version:      version,
		log:          log,
	}
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

func (s *service) setupInternalDirectories(testDir string, params types.RunParams, genesis *core.Genesis, snapshot *benchmark.SnapshotDefinition, role string) (*config.InternalClientOptions, error) {
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create test directory")
	}

	metricsPath := path.Join(testDir, "metrics")
	err = os.Mkdir(metricsPath, 0755)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create metrics directory")
	}

	// write chain config to testDir/chain.json
	chainCfgPath := path.Join(testDir, "chain.json")
	chainCfgFile, err := os.OpenFile(chainCfgPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open chain config file")
	}

	// write chain cfg
	err = json.NewEncoder(chainCfgFile).Encode(genesis)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write chain config")
	}

	var dataDirPath string
	isSnapshot := snapshot != nil && snapshot.Command != ""
	if isSnapshot {
		// if we have a snapshot, restore it if needed or reuse from a previous test
		snapshotDir, err := s.dataDirState.EnsureSnapshot(*snapshot, params.NodeType, role)
		if err != nil {
			return nil, errors.Wrap(err, "failed to ensure snapshot")
		}

		dataDirPath = snapshotDir
	} else {
		// if no snapshot, just create a new datadir
		dataDirPath = path.Join(testDir, "data")
		err = os.Mkdir(dataDirPath, 0755)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create data directory")
		}
	}

	var jwtSecret [32]byte
	_, err = rand.Read(jwtSecret[:])
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate jwt secret")
	}

	jwtSecretPath := path.Join(testDir, "jwt_secret")
	jwtSecretFile, err := os.OpenFile(jwtSecretPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open jwt secret file")
	}

	_, err = jwtSecretFile.Write([]byte(hex.EncodeToString(jwtSecret[:])))
	if err != nil {
		return nil, errors.Wrap(err, "failed to write jwt secret")
	}

	if err = jwtSecretFile.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close jwt secret file")
	}

	options := s.config.ClientOptions()
	options = params.ClientOptions(options)

	options.SkipInit = isSnapshot

	internalOptions := &config.InternalClientOptions{
		ClientOptions: options,
		JWTSecretPath: jwtSecretPath,
		MetricsPath:   metricsPath,
		JWTSecret:     hex.EncodeToString(jwtSecret[:]),
		ChainCfgPath:  chainCfgPath,
		DataDirPath:   dataDirPath,
		TestDirPath:   testDir,
	}

	return internalOptions, nil
}

type TestRunMetadata struct {
	TestName string  `json:"test_name"`
	Success  bool    `json:"success"`
	Error    *string `json:"error,omitempty"`
}

func (s *service) exportOutput(testName string, returnedError error, testDirs *config.InternalClientOptions, testOutputDir string, nodeType string) error {
	// package up logs from the EL client and write them to the output dir
	// outputDir/
	//  ├── <testName>
	//  │   ├── result-<node_type>.json
	//  │   ├── logs-<node_type>.gz
	//  │   ├── metrics-<node_type>.json

	// create output directory

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

func (s *service) setupDataDirs(workingDir string, params types.RunParams, genesis *core.Genesis, snapshot *benchmark.SnapshotDefinition) (*config.InternalClientOptions, *config.InternalClientOptions, error) {
	// create temp directory for this test
	testName := fmt.Sprintf("%d-%s-test", time.Now().Unix(), params.NodeType)
	sequencerTestDir := path.Join(workingDir, fmt.Sprintf("%s-sequencer", testName))
	validatorTestDir := path.Join(workingDir, fmt.Sprintf("%s-validator", testName))

	sequencerOptions, err := s.setupInternalDirectories(sequencerTestDir, params, genesis, snapshot, "sequencer")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to setup internal directories")
	}

	validatorOptions, err := s.setupInternalDirectories(validatorTestDir, params, genesis, snapshot, "validator")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to setup internal directories")
	}

	return sequencerOptions, validatorOptions, nil
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

func (s *service) runTest(ctx context.Context, params types.RunParams, workingDir string, outputDir string, snapshotConfig *benchmark.SnapshotDefinition, proofConfig *benchmark.ProofProgramOptions, transactionPayload payload.Definition) (*benchmark.RunResult, error) {

	s.log.Info(fmt.Sprintf("Running benchmark with params: %+v", params))

	// get genesis block
	genesis, err := s.getGenesisForSnapshotConfig(snapshotConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get genesis block")
	}

	// create temp directory for this test
	testName := fmt.Sprintf("%d-%s-test", time.Now().Unix(), params.NodeType)
	sequencerTestDir := path.Join(workingDir, fmt.Sprintf("%s-sequencer", testName))
	validatorTestDir := path.Join(workingDir, fmt.Sprintf("%s-validator", testName))

	// setup data directories (restore from snapshot if needed)
	sequencerOptions, validatorOptions, err := s.setupDataDirs(workingDir, params, genesis, snapshotConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup data dirs")
	}

	if proofConfig != nil {
		if err := s.setupBlobsDir(workingDir); err != nil {
			return nil, errors.Wrap(err, "failed to setup blobs directory")
		}
	}

	defer func() {
		// clean up test directory
		err := os.RemoveAll(sequencerTestDir)
		if err != nil {
			log.Error("failed to remove test directory", "err", err)
		}

		// clean up test directory
		err = os.RemoveAll(validatorTestDir)
		if err != nil {
			log.Error("failed to remove test directory", "err", err)
		}
	}()

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
	benchmark, err := network.NewNetworkBenchmark(config, s.log, sequencerOptions, validatorOptions, proofConfig, transactionPayload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create network benchmark")
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

	metadata := benchmark.RunGroupFromTestPlans(testPlans)
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

			metricSummary, err := s.runTest(ctx, c.Params, s.config.DataDir(), outputDir, testPlan.Snapshot, testPlan.ProofProgram, transactionPayloads[c.Params.PayloadID])
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
