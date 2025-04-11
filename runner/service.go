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
	"github.com/base/base-bench/runner/config"
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

func (s *service) setupInternalDirectories(testDir string, params benchmark.Params, genesis *core.Genesis) (*config.InternalClientOptions, error) {
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
		return nil, errors.Wrap(err, "failed to close jwt secret file")
	}

	options := s.config.ClientOptions()
	options = params.ClientOptions(options)

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

const (
	ResultMetadataFileName = "result.json"
	CompressedLogsFileName = "logs.gz"
)

func (s *service) exportOutput(testName string, returnedError error, testDirs *config.InternalClientOptions, testOutputDir string) error {
	// package up logs from the EL client and write them to the output dir
	// outputDir/
	//  ├── <testName>
	//  │   ├── result.json
	//  │   ├── logs.gz
	//  │   ├── metrics.json

	// create output directory

	// copy metrics.json to output dir
	metricsPath := path.Join(testDirs.MetricsPath, metrics.MetricsFileName)
	metricsOutputPath := path.Join(testOutputDir, metrics.MetricsFileName)
	err := os.Rename(metricsPath, metricsOutputPath)
	if err != nil {
		return errors.Wrap(err, "failed to move metrics file")
	}

	// copy logs to output dir gzipped
	logsPath := path.Join(testDirs.TestDirPath, network.ExecutionLayerLogFileName)
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

func (s *service) runTest(ctx context.Context, params benchmark.Params, rootDir string) error {
	s.log.Info(fmt.Sprintf("Running benchmark with params: %+v", params))

	// for devnets, just create a new genesis with the current time
	genesisTime := time.Now()
	genesis := params.Genesis(genesisTime)

	// create temp directory for this test
	testName := fmt.Sprintf("%d-%s-test", time.Now().Unix(), params.NodeType)
	sequencerTestDir := path.Join(rootDir, fmt.Sprintf("%s-sequencer", testName))
	validatorTestDir := path.Join(rootDir, fmt.Sprintf("%s-validator", testName))

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

	sequencerOptions, err := s.setupInternalDirectories(sequencerTestDir, params, &genesis)
	if err != nil {
		return errors.Wrap(err, "failed to setup internal directories")
	}

	validatorOptions, err := s.setupInternalDirectories(validatorTestDir, params, &genesis)
	if err != nil {
		return errors.Wrap(err, "failed to setup internal directories")
	}

	// Run benchmark
	benchmark, err := network.NewNetworkBenchmark(s.log, params, sequencerOptions, validatorOptions, &genesis, s.config)
	if err != nil {
		return errors.Wrap(err, "failed to create network benchmark")
	}
	err = benchmark.Run(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to export output")
	}

	// TODO: add test name from YAML in a nice format
	sequencerOutputDir := path.Join(s.config.OutputDir(), fmt.Sprintf("%s-sequencer", params.NodeType))
	validatorOutputDir := path.Join(s.config.OutputDir(), fmt.Sprintf("%s-validator", params.NodeType))
	for _, dir := range []string{sequencerOutputDir, validatorOutputDir} {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return errors.Wrap(err, "failed to create output directory")
		}
	}

	err = s.exportOutput(testName, err, sequencerOptions, sequencerOutputDir)
	if err != nil {
		return errors.Wrap(err, "failed to export output")
	}

	err = s.exportOutput(testName, err, validatorOptions, validatorOutputDir)
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
			err = s.runTest(ctx, params, dataDir)
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
