package network

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"path"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/clients"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"

	"github.com/base/base-bench/runner/logger"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network/proofprogram"
	"github.com/base/base-bench/runner/network/proofprogram/fakel1"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

const (
	ExecutionLayerLogFileName = "el.log"
)

type TestConfig struct {
	Params     benchmark.Params
	Config     config.Config
	Genesis    core.Genesis
	BatcherKey ecdsa.PrivateKey
	// BatcherAddr is lazily initialized to avoid unnecessary computation.
	batcherAddr *common.Address
}

func (c *TestConfig) BatcherAddr() common.Address {
	if c.batcherAddr == nil {
		batcherAddr := crypto.PubkeyToAddress(c.BatcherKey.PublicKey)
		c.batcherAddr = &batcherAddr
	}
	return *c.batcherAddr
}

// NetworkBenchmark handles the lifecycle for a single benchmark run.
type NetworkBenchmark struct {
	log log.Logger

	sequencerOptions *config.InternalClientOptions
	validatorOptions *config.InternalClientOptions

	collectedSequencerMetrics *benchmark.SequencerKeyMetrics
	collectedValidatorMetrics *benchmark.ValidatorKeyMetrics

	testConfig  *TestConfig
	proofConfig *benchmark.ProofProgramOptions
}

// NewNetworkBenchmark creates a new network benchmark and initializes the payload worker and consensus client.
func NewNetworkBenchmark(config *TestConfig, log log.Logger, sequencerOptions *config.InternalClientOptions, validatorOptions *config.InternalClientOptions, proofConfig *benchmark.ProofProgramOptions) (*NetworkBenchmark, error) {
	return &NetworkBenchmark{
		log:              log,
		sequencerOptions: sequencerOptions,
		validatorOptions: validatorOptions,
		testConfig:       config,
		proofConfig:      proofConfig,
	}, nil
}

func (nb *NetworkBenchmark) Run(ctx context.Context) (err error) {
	l1Chain, err := newL1Chain(nb.testConfig)
	if err != nil {
		return fmt.Errorf("failed to create L1 chain: %w", err)
	}

	payloads, firstTestBlock, err := nb.benchmarkSequencer(ctx, l1Chain)
	if err != nil {
		return fmt.Errorf("failed to run sequencer: %w", err)
	}
	err = nb.benchmarkValidator(ctx, payloads, firstTestBlock, l1Chain, &nb.testConfig.BatcherKey)
	if err != nil {
		return fmt.Errorf("failed to run validator: %w", err)
	}
	return nil
}

func (nb *NetworkBenchmark) benchmarkFaultProofProgram(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64, l2RPCURL string, l1Chain *fakel1.FakeL1Chain, batcherKey *ecdsa.PrivateKey) error {
	if nb.proofConfig == nil {
		nb.log.Info("Skipping fault proof program benchmark as it is not enabled")
		return nil
	}

	version := nb.proofConfig.Version
	if version == "" {
		return fmt.Errorf("proof_program.version is not set")
	}

	// ensure binary exists
	binaryPath := path.Join("op-program", "versions", version, "op-program")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("proof program binary does not exist at %s", binaryPath)
	}

	opProgram := proofprogram.NewOPProgram(&nb.testConfig.Genesis, nb.log, binaryPath, l2RPCURL, l1Chain, batcherKey)

	return opProgram.Run(ctx, payloads, firstTestBlock)
}

func (nb *NetworkBenchmark) benchmarkSequencer(ctx context.Context, l1Chain *l1Chain) ([]engine.ExecutableData, uint64, error) {
	sequencerClient, err := setupNode(ctx, nb.log, nb.testConfig.Params, nb.sequencerOptions)
	if err != nil {
		return nil, 0, err
	}

	defer sequencerClient.Stop()

	// Create metrics collector and writer
	metricsCollector := metrics.NewMetricsCollector(nb.log, sequencerClient.Client(), nb.testConfig.Params.NodeType, sequencerClient.MetricsPort())
	metricsWriter := metrics.NewFileMetricsWriter(nb.sequencerOptions.MetricsPath)

	defer func() {
		sequencerMetrics := metricsCollector.GetMetrics()

		nb.collectedSequencerMetrics = metrics.BlockMetricsToSequencerSummary(sequencerMetrics)

		if err := metricsWriter.Write(sequencerMetrics); err != nil {
			nb.log.Error("Failed to write metrics", "error", err)
		}
	}()

	benchmark := newSequencerBenchmark(nb.log, *nb.testConfig, sequencerClient, l1Chain)

	return benchmark.Run(ctx, metricsCollector)
}

func (nb *NetworkBenchmark) benchmarkValidator(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64, l1Chain *l1Chain, batcherKey *ecdsa.PrivateKey) error {
	validatorClient, err := setupNode(ctx, nb.log, nb.testConfig.Params, nb.validatorOptions)
	if err != nil {
		return err
	}

	defer validatorClient.Stop()

	// Create metrics collector and writer
	metricsCollector := metrics.NewMetricsCollector(nb.log, validatorClient.Client(), nb.testConfig.Params.NodeType, validatorClient.MetricsPort())
	metricsWriter := metrics.NewFileMetricsWriter(nb.validatorOptions.MetricsPath)

	defer func() {
		validatorMetrics := metricsCollector.GetMetrics()

		nb.collectedValidatorMetrics = metrics.BlockMetricsToValidatorSummary(validatorMetrics)

		if err := metricsWriter.Write(validatorMetrics); err != nil {
			nb.log.Error("Failed to write metrics", "error", err)
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
