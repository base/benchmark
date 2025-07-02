package consensus

import (
	"context"
	"math/big"
	"time"

	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// SyncingConsensusClient is a fake consensus client that generates blocks on a timer.
type SyncingConsensusClient struct {
	*BaseConsensusClient
}

// NewSyncingConsensusClient creates a new consensus client.
func NewSyncingConsensusClient(log log.Logger, client *ethclient.Client, authClient client.RPC, options ConsensusClientOptions, headBlockHash common.Hash, headBlockNumber uint64) *SyncingConsensusClient {
	base := NewBaseConsensusClient(log, client, authClient, options, headBlockHash, headBlockNumber)
	return &SyncingConsensusClient{
		BaseConsensusClient: base,
	}
}

// Propose starts block generation, waits BlockTime, and generates a block.
func (f *SyncingConsensusClient) propose(ctx context.Context, payload *engine.ExecutableData, blockMetrics *metrics.BlockMetrics) error {

	root := crypto.Keccak256Hash([]byte("fake-beacon-block-root"), big.NewInt(1).Bytes())

	f.log.Info("Validate payload", "payload_index", payload.Number)
	startTime := time.Now()
	err := f.newPayload(ctx, payload, root)
	if err != nil {
		return err
	}

	f.headBlockHash = payload.BlockHash
	duration := time.Since(startTime)
	f.log.Info("Validated payload", "payload_index", payload.Number, "duration", duration)
	blockMetrics.AddExecutionMetric(types.NewPayloadLatencyMetric, duration)

	// fetch gas used from the payload
	gasUsed := payload.GasUsed
	gasPerSecond := float64(gasUsed) / duration.Seconds()

	blockMetrics.AddExecutionMetric(types.GasPerBlockMetric, float64(gasUsed))
	blockMetrics.AddExecutionMetric(types.GasPerSecondMetric, gasPerSecond)

	startTime = time.Now()
	_, err = f.updateForkChoice(ctx, nil)
	if err != nil {
		return err
	}
	duration = time.Since(startTime)
	blockMetrics.AddExecutionMetric(types.UpdateForkChoiceLatencyMetric, duration)

	return nil
}

// Start starts the fake consensus client.
func (f *SyncingConsensusClient) Start(ctx context.Context, payloads []engine.ExecutableData, metricsCollector metrics.Collector, firstTestBlock uint64) error {
	f.log.Info("Starting sync benchmark", "num_payloads", len(payloads))
	m := metrics.NewBlockMetrics()
	for i := 0; i < len(payloads); i++ {
		m.SetBlockNumber(uint64(max(0, int(payloads[i].Number)-int(firstTestBlock))))
		f.log.Info("Proposing payload", "payload_index", i)
		err := f.propose(ctx, &payloads[i], m)
		if err != nil {
			return err
		}

		if payloads[i].Number >= firstTestBlock {
			err = metricsCollector.Collect(ctx, m)
			if err != nil {
				f.log.Error("Failed to collect metrics", "error", err)
			}
		}
	}
	return nil
}
