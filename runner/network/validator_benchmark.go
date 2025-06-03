package network

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"path"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network/consensus"
	"github.com/base/base-bench/runner/network/proofprogram"
	"github.com/base/base-bench/runner/network/proofprogram/fakel1"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

type validatorBenchmark struct {
	log             log.Logger
	validatorClient types.ExecutionClient
	config          TestConfig
	proofConfig     *benchmark.ProofProgramOptions
	l1Chain         *l1Chain
}

func newValidatorBenchmark(log log.Logger, config TestConfig, validatorClient types.ExecutionClient, l1Chain *l1Chain, proofConfig *benchmark.ProofProgramOptions) *validatorBenchmark {
	return &validatorBenchmark{
		log:             log,
		config:          config,
		validatorClient: validatorClient,
		proofConfig:     proofConfig,
		l1Chain:         l1Chain,
	}
}

func (vb *validatorBenchmark) benchmarkFaultProofProgram(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64, l1Chain *fakel1.FakeL1Chain, batcherKey *ecdsa.PrivateKey) error {
	if vb.proofConfig == nil {
		vb.log.Info("Skipping fault proof program benchmark as it is not enabled")
		return nil
	}

	version := vb.proofConfig.Version
	if version == "" {
		return fmt.Errorf("proof_program.version is not set")
	}

	// ensure binary exists
	binaryPath := path.Join("op-program", "versions", version, "op-program")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("proof program binary does not exist at %s", binaryPath)
	}

	opProgram := proofprogram.NewOPProgram(&vb.config.Genesis, vb.log, binaryPath, vb.validatorClient.ClientURL(), l1Chain, batcherKey)

	return opProgram.Run(ctx, payloads, firstTestBlock)
}

func (vb *validatorBenchmark) Run(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64, metricsCollector metrics.MetricsCollector) error {
	headBlockHeader, err := vb.validatorClient.Client().HeaderByNumber(ctx, nil)
	if err != nil {
		vb.log.Warn("failed to get head block header", "err", err)
		return err
	}
	headBlockHash := headBlockHeader.Hash()
	headBlockNumber := headBlockHeader.Number.Uint64()

	consensusClient := consensus.NewSyncingConsensusClient(vb.log, vb.validatorClient.Client(), vb.validatorClient.AuthClient(), consensus.ConsensusClientOptions{
		BlockTime: vb.config.Params.BlockTime,
	}, headBlockHash, headBlockNumber)

	err = consensusClient.Start(ctx, payloads, metricsCollector, firstTestBlock)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		vb.log.Warn("failed to run consensus client", "err", err)
		return err
	}

	err = vb.benchmarkFaultProofProgram(ctx, payloads, firstTestBlock, vb.l1Chain.chain, &vb.config.BatcherKey)
	if err != nil {
		return fmt.Errorf("failed to run fault proof program: %w", err)
	}

	return nil
}
