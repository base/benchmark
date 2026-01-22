package network

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/benchmark/portmanager"
	"github.com/base/base-bench/runner/clients"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/network/flashblocks"
	"github.com/base/base-bench/runner/payload"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/base/base-bench/runner/logger"
	"github.com/base/base-bench/runner/metrics"

	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

const (
	ExecutionLayerLogFileName = "el.log"
)

// NetworkBenchmark handles the lifecycle for a single benchmark run
type NetworkBenchmark struct {
	log log.Logger

	sequencerOptions *config.InternalClientOptions
	validatorOptions *config.InternalClientOptions

	collectedSequencerMetrics *benchtypes.SequencerKeyMetrics
	collectedValidatorMetrics *benchtypes.ValidatorKeyMetrics

	testConfig  *benchtypes.TestConfig
	proofConfig *benchmark.ProofProgramOptions

	transactionPayload payload.Definition
	ports              portmanager.PortManager
}

// NewNetworkBenchmark creates a new network benchmark and initializes the payload worker and consensus client
func NewNetworkBenchmark(config *benchtypes.TestConfig, log log.Logger, sequencerOptions *config.InternalClientOptions, validatorOptions *config.InternalClientOptions, proofConfig *benchmark.ProofProgramOptions, transactionPayload payload.Definition, ports portmanager.PortManager) (*NetworkBenchmark, error) {
	return &NetworkBenchmark{
		log:                log,
		sequencerOptions:   sequencerOptions,
		validatorOptions:   validatorOptions,
		testConfig:         config,
		proofConfig:        proofConfig,
		transactionPayload: transactionPayload,
		ports:              ports,
	}, nil
}

// Run executes the benchmark test
func (nb *NetworkBenchmark) Run(ctx context.Context) error {
	// Create an L1 chain if needed for fault proof benchmark
	var l1Chain *l1Chain
	if nb.proofConfig != nil {
		var err error
		l1Chain, err = newL1Chain(nb.testConfig)
		if err != nil {
			return fmt.Errorf("failed to create L1 chain: %w", err)
		}
	}

	// Benchmark the sequencer first to build payloads
	payloadResult, lastSetupBlock, sequencerClient, err := nb.benchmarkSequencer(ctx, l1Chain)
	if err != nil {
		return fmt.Errorf("failed to run sequencer benchmark: %w", err)
	}

	// Benchmark the validator to sync the payloads
	if err := nb.benchmarkValidator(ctx, payloadResult, lastSetupBlock, l1Chain, sequencerClient); err != nil {
		return fmt.Errorf("failed to run validator benchmark: %w", err)
	}

	return nil
}

func (nb *NetworkBenchmark) benchmarkSequencer(ctx context.Context, l1Chain *l1Chain) (*benchtypes.PayloadResult, uint64, types.ExecutionClient, error) {
	sequencerClient, err := setupNode(ctx, nb.log, nb.testConfig.Params.NodeType, nb.testConfig.Params, nb.sequencerOptions, nb.ports, "")
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to setup sequencer node: %w", err)
	}

	// Create metrics collector and writer
	metricsCollector := sequencerClient.MetricsCollector()
	metricsWriter := metrics.NewFileMetricsWriter(nb.sequencerOptions.MetricsPath)

	// Collect metrics in a deferred function to ensure they're always collected
	defer func() {
		sequencerMetrics := metricsCollector.GetMetrics()
		if sequencerMetrics != nil {
			nb.collectedSequencerMetrics = benchtypes.BlockMetricsToSequencerSummary(sequencerMetrics)
			if err := metricsWriter.Write(sequencerMetrics); err != nil {
				nb.log.Error("Failed to write sequencer metrics", "error", err)
			}
		}
	}()

	benchmark := newSequencerBenchmark(nb.log, *nb.testConfig, sequencerClient, l1Chain, nb.transactionPayload)
	payloadResult, lastBlock, err := benchmark.Run(ctx, metricsCollector)

	if err != nil {
		sequencerClient.Stop()
		return nil, 0, nil, fmt.Errorf("failed to run sequencer benchmark: %w", err)
	}

	return payloadResult, lastBlock, sequencerClient, nil
}

func (nb *NetworkBenchmark) benchmarkValidator(ctx context.Context, payloadResult *benchtypes.PayloadResult, lastSetupBlock uint64, l1Chain *l1Chain, sequencerClient types.ExecutionClient) error {
	payloads := payloadResult.ExecutablePayloads

	var flashblockServer *flashblocks.ReplayServer
	var flashblockServerURL string

	if payloadResult.HasFlashblocks() {
		flashblockPort := nb.ports.AcquirePort("flashblocks", portmanager.FlashblocksWebsocketPortPurpose)

		flashblockServer = flashblocks.NewReplayServer(
			nb.log,
			flashblockPort,
			payloadResult.Flashblocks,
			nb.testConfig.Params.BlockTime,
		)

		if err := flashblockServer.Start(ctx); err != nil {
			nb.ports.ReleasePort(flashblockPort)
			sequencerClient.Stop()
			return fmt.Errorf("failed to start flashblock replay server: %w", err)
		}

		flashblockServerURL = flashblockServer.URL()
		nb.log.Info("Started flashblock replay server", "url", flashblockServerURL, "num_flashblocks", len(payloadResult.Flashblocks))

		defer func() {
			if err := flashblockServer.Stop(); err != nil {
				nb.log.Warn("Failed to stop flashblock replay server", "err", err)
			}
			nb.ports.ReleasePort(flashblockPort)
		}()
	}

	// Use ValidatorNodeType if specified, otherwise fall back to NodeType
	validatorNodeType := nb.testConfig.Params.ValidatorNodeType
	if validatorNodeType == "" {
		validatorNodeType = nb.testConfig.Params.NodeType
	}

	validatorClient, err := setupNode(ctx, nb.log, validatorNodeType, nb.testConfig.Params, nb.validatorOptions, nb.ports, flashblockServerURL)
	if err != nil {
		sequencerClient.Stop()
		return fmt.Errorf("failed to setup validator node: %w", err)
	}

	defer func() {
		currentHeader, err := validatorClient.Client().HeaderByNumber(ctx, nil)
		if err != nil {
			nb.log.Error("Failed to get current block number", "error", err)
		} else {
			nb.log.Info("Validator node stopped at block", "number", currentHeader.Number.Uint64(), "hash", currentHeader.Hash().Hex())
		}
		validatorClient.Stop()
	}()

	// check if validator is behind first test block
	validatorHeader, err := validatorClient.Client().HeaderByNumber(ctx, nil)
	if err != nil {
		sequencerClient.Stop()
		return fmt.Errorf("failed to get validator header: %w", err)
	}

	nb.log.Info("Validator header", "number", validatorHeader.Number.Uint64(), "lastSetupBlock", lastSetupBlock)

	if validatorHeader.Number.Cmp(big.NewInt(int64(lastSetupBlock))) < 0 {
		nb.log.Info("Validator is behind first test block, catching up", "validator_block", validatorHeader.Number.Uint64(), "last_setup_block", lastSetupBlock)
		// fetch all blocks the validator node is missing
		for i := validatorHeader.Number.Uint64() + 1; i <= lastSetupBlock; i++ {
			block, err := sequencerClient.Client().BlockByNumber(ctx, big.NewInt(int64(i)))
			if err != nil {
				sequencerClient.Stop()
				return fmt.Errorf("failed to get block %d: %w", i, err)
			}

			log.Info("Sending newpayload to validator node to catch up", "block", block.NumberU64(), "withdrawalsRoot", block.WithdrawalsRoot())

			// send newpayload to validator node
			payload := engine.BlockToExecutableData(block, big.NewInt(0), []*ethTypes.BlobTxSidecar{}, [][]byte{}).ExecutionPayload
			payload.WithdrawalsRoot = block.WithdrawalsRoot()
			root := crypto.Keccak256Hash([]byte("fake-beacon-block-root"), big.NewInt(int64(1)).Bytes())

			err = validatorClient.AuthClient().CallContext(ctx, nil, "engine_newPayloadV4", payload, []common.Hash{}, root, []common.Hash{})
			if err != nil {
				validatorClient.Stop()
				return fmt.Errorf("failed to send newpayload to validator node: %w", err)
			}

			forkchoiceUpdate := engine.ForkchoiceStateV1{
				HeadBlockHash:      payload.BlockHash,
				SafeBlockHash:      payload.BlockHash,
				FinalizedBlockHash: payload.BlockHash,
			}

			err = validatorClient.AuthClient().CallContext(ctx, nil, "engine_forkchoiceUpdatedV3", forkchoiceUpdate, nil)
			if err != nil {
				validatorClient.Stop()
				return fmt.Errorf("failed to send forkchoice update to validator node: %w", err)
			}
		}
	}
	sequencerClient.Stop()

	// Create metrics collector and writer
	metricsCollector := validatorClient.MetricsCollector()
	metricsWriter := metrics.NewFileMetricsWriter(nb.validatorOptions.MetricsPath)

	// Collect metrics in a deferred function to ensure they're always collected
	defer func() {
		validatorMetrics := metricsCollector.GetMetrics()
		if validatorMetrics != nil {
			nb.collectedValidatorMetrics = benchtypes.BlockMetricsToValidatorSummary(validatorMetrics)
			if err := metricsWriter.Write(validatorMetrics); err != nil {
				nb.log.Error("Failed to write validator metrics", "error", err)
			}
		}
	}()

	benchmark := newValidatorBenchmark(nb.log, *nb.testConfig, validatorClient, l1Chain, nb.proofConfig, flashblockServer)
	return benchmark.Run(ctx, payloads, lastSetupBlock, metricsCollector)
}

func (nb *NetworkBenchmark) GetResult() (*benchmark.RunResult, error) {
	if nb.collectedSequencerMetrics == nil || nb.collectedValidatorMetrics == nil {
		return nil, errors.New("metrics not collected")
	}

	return &benchmark.RunResult{
		SequencerMetrics: *nb.collectedSequencerMetrics,
		ValidatorMetrics: *nb.collectedValidatorMetrics,
		Success:          true,
		Complete:         true,
	}, nil
}

func setupNode(ctx context.Context, l log.Logger, nodeTypeStr string, params benchtypes.RunParams, options *config.InternalClientOptions, portManager portmanager.PortManager, flashblockServerURL string) (types.ExecutionClient, error) {
	if options == nil {
		return nil, errors.New("client options cannot be nil")
	}

	var nodeType clients.Client
	switch nodeTypeStr {
	case "geth":
		nodeType = clients.Geth
	case "reth":
		nodeType = clients.Reth
	case "rbuilder":
		nodeType = clients.Rbuilder
	case "base-reth-node":
		nodeType = clients.BaseRethNode
	default:
		return nil, fmt.Errorf("unsupported node type: %s", nodeTypeStr)
	}

	clientLogger := l.With("nodeType", nodeTypeStr)
	client := clients.NewClient(nodeType, clientLogger, options, portManager)

	logPath := path.Join(options.TestDirPath, ExecutionLayerLogFileName)
	fileWriter, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file at %s: %w", logPath, err)
	}

	stdoutLogger := logger.NewMultiWriterCloser(logger.NewLogWriter(clientLogger), fileWriter)
	stderrLogger := logger.NewMultiWriterCloser(logger.NewLogWriter(clientLogger), fileWriter)

	args := make([]string, len(options.NodeArgs))
	copy(args, options.NodeArgs)

	var flashblocksURLPtr *string
	if flashblockServerURL != "" {
		flashblocksURLPtr = &flashblockServerURL
	}

	runtimeConfig := &types.RuntimeConfig{
		Stdout:         stdoutLogger,
		Stderr:         stderrLogger,
		Args:           args,
		FlashblocksURL: flashblocksURLPtr,
	}

	if err := client.Run(ctx, runtimeConfig); err != nil {
		return nil, fmt.Errorf("failed to run execution layer client: %w", err)
	}

	return client, nil
}
