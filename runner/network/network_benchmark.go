package network

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/clients"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"

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

	collectedSequencerMetrics *benchmark.SequencerKeyMetrics
	collectedValidatorMetrics *benchmark.ValidatorKeyMetrics

	testConfig  *benchtypes.TestConfig
	proofConfig *benchmark.ProofProgramOptions
}

// NewNetworkBenchmark creates a new network benchmark and initializes the payload worker and consensus client
func NewNetworkBenchmark(config *benchtypes.TestConfig, log log.Logger, sequencerOptions *config.InternalClientOptions, validatorOptions *config.InternalClientOptions, proofConfig *benchmark.ProofProgramOptions) (*NetworkBenchmark, error) {
	return &NetworkBenchmark{
		log:              log,
		sequencerOptions: sequencerOptions,
		validatorOptions: validatorOptions,
		testConfig:       config,
		proofConfig:      proofConfig,
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
	sequencerClient, err := setupNode(ctx, nb.log, nb.testConfig.Params, nb.sequencerOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to setup sequencer node: %w", err)
	}

	// Ensure client is stopped even if benchmark fails
	defer sequencerClient.Stop()

	// Create metrics collector and writer
	metricsCollector := metrics.NewMetricsCollector(nb.log, sequencerClient.Client(), nb.testConfig.Params.NodeType, sequencerClient.MetricsPort())
	metricsWriter := metrics.NewFileMetricsWriter(nb.sequencerOptions.MetricsPath)

	// Collect metrics in a deferred function to ensure they're always collected
	defer func() {
		sequencerMetrics := metricsCollector.GetMetrics()
		if sequencerMetrics != nil {
			nb.collectedSequencerMetrics = metrics.BlockMetricsToSequencerSummary(sequencerMetrics)
			if err := metricsWriter.Write(sequencerMetrics); err != nil {
				nb.log.Error("Failed to write sequencer metrics", "error", err)
			}
		}
	}()

	benchmark := newSequencerBenchmark(nb.log, *nb.testConfig, sequencerClient, l1Chain)
	return benchmark.Run(ctx, metricsCollector)
}

func (nb *NetworkBenchmark) benchmarkValidator(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64, l1Chain *l1Chain) error {
	validatorClient, err := setupNode(ctx, nb.log, nb.testConfig.Params, nb.validatorOptions)
	if err != nil {
		return fmt.Errorf("failed to setup validator node: %w", err)
	}

	// Ensure client is stopped even if benchmark fails
	defer validatorClient.Stop()

	// Create metrics collector and writer
	metricsCollector := metrics.NewMetricsCollector(nb.log, validatorClient.Client(), nb.testConfig.Params.NodeType, validatorClient.MetricsPort())
	metricsWriter := metrics.NewFileMetricsWriter(nb.validatorOptions.MetricsPath)

	// Collect metrics in a deferred function to ensure they're always collected
	defer func() {
		validatorMetrics := metricsCollector.GetMetrics()
		if validatorMetrics != nil {
			nb.collectedValidatorMetrics = metrics.BlockMetricsToValidatorSummary(validatorMetrics)
			if err := metricsWriter.Write(validatorMetrics); err != nil {
				nb.log.Error("Failed to write validator metrics", "error", err)
			}
		}
	}()

	benchmark := newValidatorBenchmark(nb.log, *nb.testConfig, validatorClient, l1Chain, nb.proofConfig)
	return benchmark.Run(ctx, payloads, firstTestBlock, metricsCollector)
}

func (nb *NetworkBenchmark) GetResult() (*benchmark.BenchmarkRunResult, error) {
	if nb.collectedSequencerMetrics == nil || nb.collectedValidatorMetrics == nil {
		return nil, errors.New("metrics not collected")
	}

	return &benchmark.BenchmarkRunResult{
		SequencerMetrics: *nb.collectedSequencerMetrics,
		ValidatorMetrics: *nb.collectedValidatorMetrics,
		Success:          true,
	}, nil
}

func setupNode(ctx context.Context, l log.Logger, params benchmark.Params, options *config.InternalClientOptions) (types.ExecutionClient, error) {
	if options == nil {
		return nil, errors.New("client options cannot be nil")
	}

	var nodeType clients.Client
	switch params.NodeType {
	case "geth":
		nodeType = clients.Geth
	case "reth":
		nodeType = clients.Reth
	default:
		return nil, fmt.Errorf("unsupported node type: %s", params.NodeType)
	}

	clientLogger := l.With("nodeType", params.NodeType)
	client := clients.NewClient(nodeType, clientLogger, options)

	logPath := path.Join(options.TestDirPath, ExecutionLayerLogFileName)
	fileWriter, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file at %s: %w", logPath, err)
	}

	stdoutLogger := logger.NewMultiWriterCloser(logger.NewLogWriter(clientLogger), fileWriter)
	stderrLogger := logger.NewMultiWriterCloser(logger.NewLogWriter(clientLogger), fileWriter)

	runtimeConfig := &types.RuntimeConfig{
		Stdout: stdoutLogger,
		Stderr: stderrLogger,
	}

	if err := client.Run(ctx, runtimeConfig); err != nil {
		return nil, fmt.Errorf("failed to run execution layer client: %w", err)
	}

	return client, nil
}
