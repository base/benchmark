package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-yaml/yaml"
	"github.com/pkg/errors"

	"github.com/base/base-bench/clients"
	"github.com/base/base-bench/clients/types"
	"github.com/base/base-bench/network"
	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/config"
)

var ErrAlreadyStopped = errors.New("already stopped")

type Service interface {
	cliapp.Lifecycle
	Kill() error
}

type service struct {
	config  config.Config
	version string
	log     log.Logger

	stopped atomic.Bool
}

func NewService(version string, cfg config.Config, log log.Logger) Service {
	return &service{
		config:  cfg,
		version: version,
		log:     log,
	}
}

func readBenchmarkConfig(path string) ([]config.BenchmarkConfig, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}

	var config []config.BenchmarkConfig
	err = yaml.NewDecoder(file).Decode(&config)
	return config, err
}

func (s *service) Start(ctx context.Context) error {
	s.log.Info("Starting")

	config, err := readBenchmarkConfig(s.config.ConfigPath())
	if err != nil {
		return errors.Wrap(err, "failed to read benchmark config")
	}

	for _, c := range config {
		matrix, err := benchmark.NewParamsMatrixFromConfig(c)
		if err != nil {
			return errors.Wrap(err, "failed to create params matrix")
		}

		rootDir := s.config.RootDir()

		for _, params := range matrix {
			s.log.Info(fmt.Sprintf("Running benchmark with params: %+v", params))

			// create temp directory for this test
			testName := fmt.Sprintf("%d-%s-test", time.Now().Unix(), params.NodeType)
			testDir := path.Join(rootDir, testName)
			err := os.Mkdir(testDir, 0755)
			if err != nil {
				return errors.Wrap(err, "failed to create test directory")
			}

			// write chain config to testDir/chain.json
			chainCfgPath := path.Join(testDir, "chain.json")
			chainCfgFile, err := os.OpenFile(chainCfgPath, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return errors.Wrap(err, "failed to open chain config file")
			}

			// write chain cfg
			chainCfg := params.ChainConfig()
			err = json.NewEncoder(chainCfgFile).Encode(chainCfg)
			if err != nil {
				return errors.Wrap(err, "failed to write chain config")
			}

			dataDirPath := path.Join(testDir, "data")
			err = os.Mkdir(dataDirPath, 0755)
			if err != nil {
				return errors.Wrap(err, "failed to create data directory")
			}

			defer func() {
				// clean up test directory
				err = os.RemoveAll(testDir)
				if err != nil {
					log.Error("failed to remove test directory", "err", err)
				}
			}()

			// TODO: serialize these nicer so we can pass them directly
			nodeType := types.Geth
			switch params.NodeType {
			case "geth":
				nodeType = types.Geth
			case "reth":
				nodeType = types.Reth
			}
			logger := s.log.With("nodeType", params.NodeType)

			options := s.config.ClientOptions()
			options = params.ClientOptions(options)

			client := clients.NewClient(nodeType, logger, &options)

			err = client.Run(chainCfgPath, dataDirPath)
			if err != nil {
				return errors.Wrap(err, "failed to start client")
			}
			time.Sleep(2 * time.Second)

			// Wait for RPC to become available
			clientRPC := client.Client()

			ready := false

			// retry for 5 seconds
			for i := 0; i < 5; i++ {
				num, err := clientRPC.BlockNumber(ctx)
				if err == nil {
					s.log.Info("RPC is available", "blockNumber", num)
					ready = true
					break
				}
				log.Debug("RPC not available yet", "err", err)
				time.Sleep(1 * time.Second)
			}

			if !ready {
				log.Error("RPC never became available")
			}

			benchmark := network.NewNetworkBenchmark()
			benchmark.Run(ctx)

			_, _ = benchmark.CollectResults()

			client.Stop()
		}

	}

	return nil
}

// Stopped returns if the service as a whole is stopped.
func (s *service) Stopped() bool {
	return s.stopped.Load()
}

// Kill is a convenience method to forcefully, non-gracefully, stop the Service.
func (s *service) Kill() error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return s.Stop(ctx)
}

// Stop fully stops the batch-submitter and all its resources gracefully. After stopping, it cannot be restarted.
// See driver.StopBatchSubmitting to temporarily stop the batch submitter.
// If the provided ctx is cancelled, the stopping is forced, i.e. the batching work is killed non-gracefully.
func (s *service) Stop(ctx context.Context) error {
	if s.stopped.Load() {
		return ErrAlreadyStopped
	}
	s.log.Info("Service stopping")

	// var result error

	// if result == nil {
	// 	s.stopped.Store(true)
	// 	s.log.Info("Service stopped")
	// }
	return nil
}
