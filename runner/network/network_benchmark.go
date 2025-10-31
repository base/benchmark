package network

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/benchmark/portmanager"
	"github.com/base/base-bench/runner/clients"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/payload"

	"github.com/base/base-bench/runner/logger"
	"github.com/base/base-bench/runner/metrics"

	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/ethereum/go-ethereum/beacon/engine"
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

	clientVersion  string
	snapshotConfig *benchmark.SnapshotDefinition
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

// SetSnapshotConfig sets the snapshot configuration for head rollback
func (nb *NetworkBenchmark) SetSnapshotConfig(config benchmark.SnapshotDefinition) {
	nb.snapshotConfig = &config
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
	payloads, firstTestBlock, err := nb.benchmarkSequencer(ctx, l1Chain)
	if err != nil {
		return fmt.Errorf("failed to run sequencer benchmark: %w", err)
	}

	// Benchmark the validator to sync the payloads
	if err := nb.benchmarkValidator(ctx, payloads, firstTestBlock, l1Chain); err != nil {
		return fmt.Errorf("failed to run validator benchmark: %w", err)
	}

	return nil
}

func (nb *NetworkBenchmark) benchmarkSequencer(ctx context.Context, l1Chain *l1Chain) ([]engine.ExecutableData, uint64, error) {
	sequencerClient, clientVersion, err := setupNode(ctx, nb.log, nb.testConfig.Params, nb.sequencerOptions, nb.ports)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to setup sequencer node: %w", err)
	}

	// Store client version from sequencer (both sequencer and validator use same client type)
	nb.clientVersion = clientVersion

	// Perform head rollback if configured
	if err := nb.performHeadRollbackIfNeeded(ctx, sequencerClient); err != nil {
		return nil, 0, fmt.Errorf("failed to perform head rollback on sequencer: %w", err)
	}

	// Ensure client is stopped even if benchmark fails
	defer func() {
		currentHeader, err := sequencerClient.Client().HeaderByNumber(ctx, nil)
		if err != nil {
			nb.log.Error("Failed to get current block number", "error", err)
		} else {
			nb.log.Info("Sequencer node stopped at block", "number", currentHeader.Number.Uint64(), "hash", currentHeader.Hash().Hex())
		}
		sequencerClient.Stop()
	}()

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
	return benchmark.Run(ctx, metricsCollector)
}

func (nb *NetworkBenchmark) benchmarkValidator(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64, l1Chain *l1Chain) error {
	validatorClient, _, err := setupNode(ctx, nb.log, nb.testConfig.Params, nb.validatorOptions, nb.ports)
	if err != nil {
		return fmt.Errorf("failed to setup validator node: %w", err)
	}

	// Perform head rollback if configured
	if err := nb.performHeadRollbackIfNeeded(ctx, validatorClient); err != nil {
		return fmt.Errorf("failed to perform head rollback on validator: %w", err)
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

	benchmark := newValidatorBenchmark(nb.log, *nb.testConfig, validatorClient, l1Chain, nb.proofConfig)
	return benchmark.Run(ctx, payloads, firstTestBlock, metricsCollector)
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
		ClientVersion:    nb.clientVersion,
	}, nil
}

func setupNode(ctx context.Context, l log.Logger, params benchtypes.RunParams, options *config.InternalClientOptions, portManager portmanager.PortManager) (types.ExecutionClient, string, error) {
	if options == nil {
		return nil, "", errors.New("client options cannot be nil")
	}

	var nodeType clients.Client
	switch params.NodeType {
	case "geth":
		nodeType = clients.Geth
	case "reth":
		nodeType = clients.Reth
	case "rbuilder":
		nodeType = clients.Rbuilder
	default:
		return nil, "", fmt.Errorf("unsupported node type: %s", params.NodeType)
	}

	clientLogger := l.With("nodeType", params.NodeType)
	client := clients.NewClient(nodeType, clientLogger, options, portManager)

	// Get client version before starting
	clientVersion, err := client.GetVersion(ctx)
	if err != nil {
		l.Warn("Failed to get client version", "error", err)
		clientVersion = "unknown"
	}
	l.Info("Client version detected", "version", clientVersion, "nodeType", params.NodeType)

	logPath := path.Join(options.TestDirPath, ExecutionLayerLogFileName)
	fileWriter, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open log file at %s: %w", logPath, err)
	}

	stdoutLogger := logger.NewMultiWriterCloser(logger.NewLogWriter(clientLogger), fileWriter)
	stderrLogger := logger.NewMultiWriterCloser(logger.NewLogWriter(clientLogger), fileWriter)

	runtimeConfig := &types.RuntimeConfig{
		Stdout: stdoutLogger,
		Stderr: stderrLogger,
		Args:   options.NodeArgs,
	}

	if err := client.Run(ctx, runtimeConfig); err != nil {
		return nil, "", fmt.Errorf("failed to run execution layer client: %w", err)
	}

	return client, clientVersion, nil
}

// SetupNodeForHeadDetection creates a temporary client for head block detection
func SetupNodeForHeadDetection(ctx context.Context, l log.Logger, nodeType string, options *config.InternalClientOptions, portManager portmanager.PortManager) (types.ExecutionClient, string, error) {
	// This is similar to setupNode but simplified for head detection only
	return setupNode(ctx, l, benchtypes.RunParams{NodeType: nodeType}, options, portManager)
}

// performHeadRollbackIfNeeded performs head rollback if the snapshot method is head_rollback
func (nb *NetworkBenchmark) performHeadRollbackIfNeeded(ctx context.Context, client types.ExecutionClient) error {
	if nb.snapshotConfig == nil {
		return nil
	}

	snapshotMethod := nb.snapshotConfig.GetSnapshotMethod()
	if snapshotMethod != benchmark.SnapshotMethodHeadRollback {
		return nil
	}

	// At this point, the rollback block should be set (either by user or auto-detected)
	if nb.snapshotConfig.RollbackBlock == nil {
		return fmt.Errorf("rollback_block not set for head_rollback method - this should have been auto-detected")
	}

	blockNumber := *nb.snapshotConfig.RollbackBlock
	nb.log.Info("Performing head rollback", "blockNumber", blockNumber, "nodeType", nb.testConfig.Params.NodeType)

	// Perform the rollback
	if err := client.SetHead(ctx, blockNumber); err != nil {
		return fmt.Errorf("failed to perform head rollback to block %d: %w", blockNumber, err)
	}

	nb.log.Info("Head rollback completed successfully", "blockNumber", blockNumber)
	return nil
}
