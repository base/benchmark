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
	"github.com/base/base-bench/runner/network/proofprogram/fakel1"
	benchtypes "github.com/base/base-bench/runner/network/types"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

type validatorBenchmark struct {
	log             log.Logger
	validatorClient types.ExecutionClient
	config          benchtypes.TestConfig
	proofConfig     *benchmark.ProofProgramOptions
	l1Chain         *l1Chain
}

func newValidatorBenchmark(log log.Logger, config benchtypes.TestConfig, validatorClient types.ExecutionClient, l1Chain *l1Chain, proofConfig *benchmark.ProofProgramOptions) *validatorBenchmark {
	return &validatorBenchmark{
		log:             log,
		config:          config,
		validatorClient: validatorClient,
		proofConfig:     proofConfig,
		l1Chain:         l1Chain,
	}
}

func (vb *validatorBenchmark) benchmarkFaultProofProgram(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64, l1Chain fakel1.L1Chain, batcherKey *ecdsa.PrivateKey) error {
	version := vb.proofConfig.Version
	if version == "" {
		return fmt.Errorf("proof_program.version is not set")
	}

	// ensure binary exists
	binaryPath := path.Join("op-program", "versions", version, "op-program")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("proof program binary does not exist at %s", binaryPath)
	}

	opProgramBenchmark := NewOPProgramBenchmark(&vb.config.Genesis, vb.log, binaryPath, vb.validatorClient.ClientURL(), l1Chain, batcherKey)

	return opProgramBenchmark.Run(ctx, payloads, firstTestBlock)
}

func (vb *validatorBenchmark) Run(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64, metricsCollector metrics.Collector) error {
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

	if vb.proofConfig == nil {
		vb.log.Info("Skipping fault proof program benchmark as it is not enabled")
		return nil
	}

	if vb.l1Chain == nil {
		return fmt.Errorf("l1 chain should be setup if fault proof program is enabled")
	}

	err = vb.benchmarkFaultProofProgram(ctx, payloads, firstTestBlock, vb.l1Chain.chain, &vb.config.BatcherKey)
	if err != nil {
		return fmt.Errorf("failed to run fault proof program: %w", err)
	}

	return nil
}
