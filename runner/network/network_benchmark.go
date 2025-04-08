package network

import (
	"context"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/clients"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/logger"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network/consensus"
	"github.com/base/base-bench/runner/payload"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
)

const (
	ExecutionLayerLogFileName = "el.log"
)

// NetworkBenchmark handles the lifecycle for a single benchmark run.
type NetworkBenchmark struct {
	log    log.Logger
	params benchmark.Params

	sequencerOptions *config.InternalClientOptions
	validatorOptions *config.InternalClientOptions

	genesis *core.Genesis
}

// NewNetworkBenchmark creates a new network benchmark and initializes the payload worker and consensus client.
func NewNetworkBenchmark(log log.Logger, benchParams benchmark.Params, sequencerOptions *config.InternalClientOptions, validatorOptions *config.InternalClientOptions, genesis *core.Genesis) (*NetworkBenchmark, error) {
	return &NetworkBenchmark{
		log:              log,
		sequencerOptions: sequencerOptions,
		validatorOptions: validatorOptions,
		genesis:          genesis,
		params:           benchParams,
	}, nil
}

func (nb *NetworkBenchmark) setupNode(ctx context.Context, l log.Logger, params benchmark.Params, options *config.InternalClientOptions) (types.ExecutionClient, error) {
	// TODO: serialize these nicer so we can pass them directly
	nodeType := clients.Geth
	switch params.NodeType {
	case "geth":
		nodeType = clients.Geth
	case "reth":
		nodeType = clients.Reth
	}
	clientLogger := l.With("nodeType", params.NodeType)

	client := clients.NewClient(nodeType, clientLogger, options)

	fileWriter, err := os.OpenFile(path.Join(options.TestDirPath, ExecutionLayerLogFileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open log file")
	}

	// wrap loggers with a file writer to output/el-log.log
	stdoutLogger := logger.NewMultiWriterCloser(logger.NewLogWriter(clientLogger), fileWriter)
	stderrLogger := logger.NewMultiWriterCloser(logger.NewLogWriter(clientLogger), fileWriter)

	runtimeConfig := &types.RuntimeConfig{
		Stdout: stdoutLogger,
		Stderr: stderrLogger,
	}

	err = client.Run(ctx, runtimeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run EL client")
	}

	return client, nil
}

func (nb *NetworkBenchmark) Run(ctx context.Context) error {
	payloads, err := nb.benchmarkSequencer(ctx)
	if err != nil {
		return err
	}
	return nb.benchmarkValidator(ctx, payloads)
}

func (nb *NetworkBenchmark) benchmarkSequencer(ctx context.Context) ([]engine.ExecutableData, error) {
	sequencerClient, err := nb.setupNode(ctx, nb.log, nb.params, nb.sequencerOptions)
	if err != nil {
		return nil, err
	}

	defer sequencerClient.Stop()

	// Create metrics collector and writer
	metricsCollector := metrics.NewMetricsCollector(nb.log, sequencerClient.Client(), nb.params.NodeType, sequencerClient.MetricsPort())
	metricsWriter := metrics.NewFileMetricsWriter(nb.sequencerOptions.MetricsPath)

	defer func() {
		if err := metricsWriter.Write(metricsCollector.GetMetrics()); err != nil {
			nb.log.Error("Failed to write metrics", "error", err)
		}
	}()

	amount := new(big.Int).Mul(big.NewInt(1e6), big.NewInt(params.Ether))

	mempool, worker, err := payload.NewTransferPayloadWorker(nb.log, sequencerClient.ClientURL(), nb.params, common.FromHex("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"), amount)
	if err != nil {
		return nil, err
	}

	benchmarkCtx, benchmarkCancel := context.WithCancel(ctx)

	errChan := make(chan error)
	payloadResult := make(chan []engine.ExecutableData)

	go func() {
		err := worker.Setup(benchmarkCtx)
		if err != nil {
			nb.log.Warn("failed to setup payload worker", "err", err)
			errChan <- err
			return
		}

		err = worker.Run(benchmarkCtx)
		if err != nil {
			nb.log.Warn("failed to start payload worker", "err", err)
			errChan <- err
			return
		}
	}()

	go func() {
		consensusClient := consensus.NewSequencerConsensusClient(nb.log, sequencerClient.Client(), sequencerClient.AuthClient(), mempool, nb.genesis, metricsCollector, consensus.ConsensusClientOptions{
			BlockTime: nb.params.BlockTime,
		})

		payloads := make([]engine.ExecutableData, 0)

		// wait 2 seconds before starting
		time.Sleep(2 * time.Second)

		// run for a few blocks
		for i := 0; i < 12; i++ {
			payload, err := consensusClient.Propose(benchmarkCtx)
			if err != nil {
				errChan <- err
				return
			}
			nb.log.Info("Proposed payload", "payload_index", i, "len", len(payloads))
			payloads = append(payloads, *payload)
		}
		payloadResult <- payloads
	}()

	select {
	case err := <-errChan:
		benchmarkCancel()
		return nil, err
	case payloads := <-payloadResult:
		benchmarkCancel()
		return payloads, nil
	}
}

func (nb *NetworkBenchmark) benchmarkValidator(ctx context.Context, payloads []engine.ExecutableData) error {
	validatorClient, err := nb.setupNode(ctx, nb.log, nb.params, nb.validatorOptions)
	if err != nil {
		return err
	}

	defer validatorClient.Stop()

	// Create metrics collector and writer
	metricsCollector := metrics.NewMetricsCollector(nb.log, validatorClient.Client(), nb.params.NodeType, validatorClient.MetricsPort())
	metricsWriter := metrics.NewFileMetricsWriter(nb.validatorOptions.MetricsPath)

	defer func() {
		if err := metricsWriter.Write(metricsCollector.GetMetrics()); err != nil {
			nb.log.Error("Failed to write metrics", "error", err)
		}
	}()

	consensusClient := consensus.NewSyncingConsensusClient(nb.log, validatorClient.Client(), validatorClient.AuthClient(), nb.genesis, metricsCollector, consensus.ConsensusClientOptions{
		BlockTime: nb.params.BlockTime,
	})

	err = consensusClient.Start(ctx, payloads)
	if err != nil && !errors.Is(err, context.Canceled) {
		nb.log.Warn("failed to run consensus client", "err", err)
		return err
	}
	return nil
}
