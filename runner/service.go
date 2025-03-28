package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-yaml/yaml"
	"github.com/pkg/errors"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/clients"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network"
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

func (s *service) runTest(ctx context.Context, params benchmark.Params, rootDir string) error {

	genesisTime := time.Now()
	s.log.Info(fmt.Sprintf("Running benchmark with params: %+v", params))

	// create temp directory for this test
	testName := fmt.Sprintf("%d-%s-test", time.Now().Unix(), params.NodeType)
	testDir := path.Join(rootDir, testName)
	err := os.Mkdir(testDir, 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create test directory")
	}

	metricsPath := path.Join(testDir, "metrics")
	err = os.Mkdir(metricsPath, 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create metrics directory")
	}

	// write chain config to testDir/chain.json
	chainCfgPath := path.Join(testDir, "chain.json")
	chainCfgFile, err := os.OpenFile(chainCfgPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open chain config file")
	}

	// write chain cfg
	genesis := params.Genesis(genesisTime)
	err = json.NewEncoder(chainCfgFile).Encode(genesis)
	if err != nil {
		return errors.Wrap(err, "failed to write chain config")
	}

	dataDirPath := path.Join(testDir, "data")
	err = os.Mkdir(dataDirPath, 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create data directory")
	}

	var jwtSecret [32]byte
	_, err = rand.Read(jwtSecret[:])
	if err != nil {
		return errors.Wrap(err, "failed to generate jwt secret")
	}

	jwtSecretPath := path.Join(testDir, "jwt_secret")
	jwtSecretFile, err := os.OpenFile(jwtSecretPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open jwt secret file")
	}

	_, err = jwtSecretFile.Write([]byte(hex.EncodeToString(jwtSecret[:])))
	if err != nil {
		return errors.Wrap(err, "failed to write jwt secret")
	}

	if err = jwtSecretFile.Close(); err != nil {
		return err
	}

	defer func() {
		// clean up test directory
		err = os.RemoveAll(testDir)
		if err != nil {
			log.Error("failed to remove test directory", "err", err)
		}
	}()

	// TODO: serialize these nicer so we can pass them directly
	nodeType := clients.Geth
	switch params.NodeType {
	case "geth":
		nodeType = clients.Geth
	case "reth":
		nodeType = clients.Reth
	}
	logger := s.log.With("nodeType", params.NodeType)

	options := s.config.ClientOptions()
	options = params.ClientOptions(options)

	clientCtx, cancelClient := context.WithCancel(ctx)
	defer cancelClient()

	client := clients.NewClient(nodeType, logger, &options)
	defer client.Stop()

	err = client.Run(clientCtx, chainCfgPath, jwtSecretPath, dataDirPath)
	if err != nil {
		return errors.Wrap(err, "failed to run EL client")
	}
	time.Sleep(2 * time.Second)

	// Create metrics collector and writer
	metricsCollector := metrics.NewMetricsCollector(logger, client.Client(), params.NodeType)
	metricsWriter := metrics.NewFileMetricsWriter(metricsPath)

	defer func() {
		if err := metricsWriter.Write(metricsCollector.GetMetrics()); err != nil {
			logger.Error("Failed to write metrics", "error", err)
		}
	}()

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
	if err != nil {
		return errors.Wrap(err, "failed to run network benchmark")
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

		rootDir := s.config.RootDir()

		for _, params := range matrix {
			err = s.runTest(ctx, params, rootDir)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Error("Failed to run test", "err", err)
				numFailure++
				continue
			}
			numSuccess++
		}
	}

	s.log.Info("Finished benchmarking", "numSuccess", numSuccess, "numFailure", numFailure)

	return nil
}
