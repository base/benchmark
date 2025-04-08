package runner

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-yaml/yaml"
	"github.com/pkg/errors"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/clients"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/logger"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network"
	"github.com/ethereum/go-ethereum/core"
)

var ErrAlreadyStopped = errors.New("already stopped")

type Service interface {
	Run(ctx context.Context) error
}

type service struct {
	config  config.Config
	version string
	log     log.Logger
}

func NewService(version string, cfg config.Config, log log.Logger) Service {
	return &service{
		config:  cfg,
		version: version,
		log:     log,
	}
}

func readBenchmarkConfig(path string) ([]benchmark.Matrix, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}

	var config []benchmark.Matrix
	err = yaml.NewDecoder(file).Decode(&config)
	return config, err
}

type testDirectories struct {
	chainCfgPath  string
	outputPath    string
	testDirPath   string
	jwtSecretPath string
	dataDirPath   string
	metricsPath   string
}

func (s *service) setupTest(ctx context.Context, params benchmark.Params, dataDir string, genesis core.Genesis) (*testDirectories, error) {
	// create temp directory for this test
	testName := fmt.Sprintf("%d-%s-test", time.Now().Unix(), params.NodeType)

	testDir := path.Join(dataDir, testName)
	err := os.Mkdir(testDir, 0755)
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

	dataDirPath := path.Join(testDir, "data")
	err = os.Mkdir(dataDirPath, 0755)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create data directory")
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
		return nil, err
	}

	// create output directory for this test at output/<testName>
	outputPath := path.Join(s.config.OutputDir(), testName)
	err = os.MkdirAll(outputPath, 0755)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create output directory")
	}

	return &testDirectories{
		chainCfgPath:  chainCfgPath,
		jwtSecretPath: jwtSecretPath,
		dataDirPath:   dataDirPath,
		metricsPath:   metricsPath,
		outputPath:    outputPath,
		testDirPath:   testDir,
	}, nil
}

type TestRunMetadata struct {
	TestName string  `json:"test_name"`
	Success  bool    `json:"success"`
	Error    *string `json:"error,omitempty"`
}

const (
	ExecutionLayerLogFileName = "el.log"
	ResultMetadataFileName    = "result.json"
	CompressedLogsFileName    = "logs.gz"
)

func (s *service) exportOutput(testName string, returnedError error, testDirs *testDirectories) error {
	// package up logs from the EL client and write them to the output dir
	// outputDir/
	//  ├── <testName>
	//  │   ├── result.json
	//  │   ├── logs.gz
	//  │   ├── metrics.json

	// create output directory
	testOutputDir := testDirs.outputPath

	// copy metrics.json to output dir
	metricsPath := path.Join(testDirs.metricsPath, metrics.MetricsFileName)
	metricsOutputPath := path.Join(testOutputDir, metrics.MetricsFileName)
	err := os.Rename(metricsPath, metricsOutputPath)
	if err != nil {
		return errors.Wrap(err, "failed to move metrics file")
	}

	// copy logs to output dir gzipped
	logsPath := path.Join(testDirs.testDirPath, ExecutionLayerLogFileName)
	logsOutputPath := path.Join(testOutputDir, CompressedLogsFileName)

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

	resultPath := path.Join(testOutputDir, ResultMetadataFileName)
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

func (s *service) runTest(ctx context.Context, testName string, params benchmark.Params, dataDir string) error {
	genesisTime := time.Now()
	s.log.Info(fmt.Sprintf("Running benchmark with params: %+v", params))

	genesis := params.Genesis(genesisTime)
	testDirs, err := s.setupTest(ctx, params, dataDir, genesis)
	if err != nil {
		return errors.Wrap(err, "failed to setup test")
	}

	defer func() {
		// clean up test directory
		err = os.RemoveAll(testDirs.dataDirPath)
		if err != nil {
			log.Error("failed to remove test directory", "err", err)
		}
	}()

	log := s.log.With("nodeType", params.NodeType)

	options := s.config.ClientOptions()
	options = params.ClientOptions(options)

	clientCtx, cancelClient := context.WithCancel(ctx)
	defer cancelClient()

	// TODO: serialize these nicer so we can pass them directly
	nodeType := clients.Geth
	switch params.NodeType {
	case "geth":
		nodeType = clients.Geth
	case "reth":
		nodeType = clients.Reth
	}

	client := clients.NewClient(nodeType, log, &options)
	defer client.Stop()

	fileWriter, err := os.OpenFile(path.Join(testDirs.testDirPath, ExecutionLayerLogFileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open log file")
	}

	// wrap loggers with a file writer to output/el-log.log
	stdoutLogger := logger.NewMultiWriterCloser(logger.NewLogWriter(log), fileWriter)
	stderrLogger := logger.NewMultiWriterCloser(logger.NewLogWriter(log), fileWriter)

	runtimeConfig := &types.RuntimeConfig{
		Stdout:        stdoutLogger,
		Stderr:        stderrLogger,
		ChainCfgPath:  testDirs.chainCfgPath,
		JwtSecretPath: testDirs.jwtSecretPath,
		DataDirPath:   testDirs.dataDirPath,
	}

	err = client.Run(clientCtx, runtimeConfig)
	if err != nil {
		return errors.Wrap(err, "failed to run EL client")
	}
	time.Sleep(2 * time.Second)

	// Create metrics collector and writer
	metricsCollector := metrics.NewMetricsCollector(log, client.Client(), params.NodeType)
	metricsWriter := metrics.NewFileMetricsWriter(testDirs.metricsPath)

	// Wait for RPC to become available
	clientRPC := client.Client()
	authClient := client.AuthClient()
	clientRPCURL := client.ClientURL()

	// Run benchmark
	benchmark, err := network.NewNetworkBenchmark(s.log, params, clientRPC, clientRPCURL, authClient, &genesis, metricsCollector)
	if err != nil {
		return errors.Wrap(err, "failed to create network benchmark")
	}
	err = benchmark.Run(clientCtx)
	benchmarkErr := err
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Warn("Failed to run benchmark", "err", err)
	}

	if err := metricsWriter.Write(metricsCollector.GetMetrics()); err != nil {
		log.Error("Failed to write metrics", "error", err)
	}

	if errors.Is(benchmarkErr, context.Canceled) {
		benchmarkErr = nil
	}

	err = s.exportOutput(testName, benchmarkErr, testDirs)
	if err != nil {
		return errors.Wrap(err, "failed to export output")
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

	for _, c := range config {
		matrix, err := benchmark.NewParamsMatrixFromConfig(c)
		if err != nil {
			return errors.Wrap(err, "failed to create params matrix")
		}

		dataDir := s.config.DataDir()
		outputDir := s.config.OutputDir()

		// ensure output directory exists
		for _, dir := range []string{dataDir, outputDir} {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				return errors.Wrap(err, "failed to create output directory")
			}
		}

		variation := 0
		for _, params := range matrix {
			testName := fmt.Sprintf("%s (%d)", c.Name, variation)
			err = s.runTest(ctx, testName, params, dataDir)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Error("Failed to run test", "err", err)
				numFailure++
				continue
			}
			variation++
			numSuccess++
		}
	}

	s.log.Info("Finished benchmarking", "numSuccess", numSuccess, "numFailure", numFailure)

	return nil
}
