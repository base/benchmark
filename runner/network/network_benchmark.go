package network

import (
	"context"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/clients"
	"github.com/base/base-bench/runner/clients/proxy"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/logger"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network/consensus"
	"github.com/base/base-bench/runner/network/mempool"
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
	config  config.Config
}

// NewNetworkBenchmark creates a new network benchmark and initializes the payload worker and consensus client.
func NewNetworkBenchmark(log log.Logger, benchParams benchmark.Params, sequencerOptions *config.InternalClientOptions, validatorOptions *config.InternalClientOptions, genesis *core.Genesis, config config.Config) (*NetworkBenchmark, error) {
	return &NetworkBenchmark{
		log:              log,
		sequencerOptions: sequencerOptions,
		validatorOptions: validatorOptions,
		genesis:          genesis,
		params:           benchParams,
		config:           config,
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
	payloads, lastSetupBlock, err := nb.benchmarkSequencer(ctx)
	if err != nil {
		return err
	}
	return nb.benchmarkValidator(ctx, payloads, lastSetupBlock)
}

func (nb *NetworkBenchmark) benchmarkSequencer(ctx context.Context) ([]engine.ExecutableData, uint64, error) {
	sequencerClient, err := nb.setupNode(ctx, nb.log, nb.params, nb.sequencerOptions)
	if err != nil {
		return nil, 0, err
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
	privateKey := common.FromHex("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	var mempool mempool.FakeMempool
	var worker payload.Worker

	payloadType := nb.params.TransactionPayload

	switch payloadType {
	case "tx-fuzz":
		proxyServer := proxy.NewProxyServer(sequencerClient, nb.log, nb.config.ProxyPort())
		err = proxyServer.Run(ctx)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed to run proxy server")
		}
		defer proxyServer.Stop()
		mempool, worker, err = payload.NewTxFuzzPayloadWorker(
			nb.log, proxyServer.ClientURL(), nb.params, privateKey, amount, nb.config.TxFuzzBinary())
	case "transfer-only":
		mempool, worker, err = payload.NewTransferPayloadWorker(
			nb.log, sequencerClient.ClientURL(), nb.params, privateKey, amount)
	default:
		return nil, 0, errors.New("invalid payload type")
	}

	if err != nil {
		return nil, 0, err
	}

	benchmarkCtx, benchmarkCancel := context.WithCancel(ctx)

	errChan := make(chan error)
	payloadResult := make(chan []engine.ExecutableData)

	setupComplete := make(chan struct{})

	go func() {
		err := worker.Setup(benchmarkCtx)
		if err != nil {
			nb.log.Warn("failed to setup payload worker", "err", err)
			errChan <- err
			return
		}
		close(setupComplete)
	}()

	var lastSetupBlock uint64

	go func() {
		consensusClient := consensus.NewSequencerConsensusClient(nb.log, sequencerClient.Client(), sequencerClient.AuthClient(), mempool, nb.genesis, consensus.ConsensusClientOptions{
			BlockTime: nb.params.BlockTime,
		})

		payloads := make([]engine.ExecutableData, 0)

		// setup blocks
		blockNum := uint64(0)

	setupLoop:
		for {
			_blockMetrics := metrics.NewBlockMetrics(blockNum)
			payload, err := consensusClient.Propose(benchmarkCtx, _blockMetrics)
			if err != nil {
				errChan <- err
				return
			}

			payloads = append(payloads, *payload)
			blockNum = payload.Number
			select {
			case <-setupComplete:
				break setupLoop
			default:
			}
		}

		lastSetupBlock = payloads[len(payloads)-1].Number
		nb.log.Info("Last setup block", "block", lastSetupBlock)

		// run for a few blocks
		for i := 0; i < nb.params.NumBlocks; i++ {
			blockMetrics := metrics.NewBlockMetrics(payloads[len(payloads)-1].Number + 1)
			err := worker.SendTxs(benchmarkCtx)
			if err != nil {
				nb.log.Warn("failed to send transactions", "err", err)
				errChan <- err
				return
			}

			payload, err := consensusClient.Propose(benchmarkCtx, blockMetrics)
			if err != nil {
				errChan <- err
				return
			}

			time.Sleep(500 * time.Millisecond)

			err = metricsCollector.Collect(benchmarkCtx, blockMetrics)
			if err != nil {
				nb.log.Error("Failed to collect metrics", "error", err)
			}
			nb.log.Info("Proposed payload", "payload_index", i, "len", len(payloads))
			payloads = append(payloads, *payload)
		}
		payloadResult <- payloads
	}()

	select {
	case err := <-errChan:
		benchmarkCancel()
		return nil, 0, err
	case payloads := <-payloadResult:
		benchmarkCancel()
		return payloads, lastSetupBlock + 1, nil
	}
}

func (nb *NetworkBenchmark) benchmarkValidator(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64) error {
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

	consensusClient := consensus.NewSyncingConsensusClient(nb.log, validatorClient.Client(), validatorClient.AuthClient(), nb.genesis, consensus.ConsensusClientOptions{
		BlockTime: nb.params.BlockTime,
	})

	err = consensusClient.Start(ctx, payloads, metricsCollector, firstTestBlock)
	if err != nil && !errors.Is(err, context.Canceled) {
		nb.log.Warn("failed to run consensus client", "err", err)
		return err
	}
	return nil
}
