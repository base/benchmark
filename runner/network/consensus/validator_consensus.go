package consensus

import (
	"context"
	"time"

	"github.com/base/base-bench/runner/metrics"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// SyncingConsensusClient is a fake consensus client that generates blocks on a timer.
type SyncingConsensusClient struct {
	*BaseConsensusClient
}

const (
	UpdateForkChoiceLatencyMetric = "latency/update_fork_choice"
	NewPayloadLatencyMetric       = "latency/new_payload"
)

// NewSyncingConsensusClient creates a new consensus client.
func NewSyncingConsensusClient(log log.Logger, client *ethclient.Client, authClient client.RPC, genesis *core.Genesis, options ConsensusClientOptions) *SyncingConsensusClient {
	base := NewBaseConsensusClient(log, client, authClient, genesis, options)
	return &SyncingConsensusClient{
		BaseConsensusClient: base,
	}
}

// Propose starts block generation, waits BlockTime, and generates a block.
func (f *SyncingConsensusClient) propose(ctx context.Context, payload *engine.ExecutableData, metrics *metrics.BlockMetrics) error {
	f.log.Info("Updating fork choice before validating payload", "payload_index", payload.Number)
	startTime := time.Now()
	_, err := f.updateForkChoice(ctx, nil)
	if err != nil {
		return err
	}
	duration := time.Since(startTime)
	metrics.AddExecutionMetric(UpdateForkChoiceLatencyMetric, duration)

	f.log.Info("Validate payload", "payload_index", payload.Number)
	startTime = time.Now()
	err = f.newPayload(ctx, payload)
	if err != nil {
		return err
	}
	duration = time.Since(startTime)
	f.log.Info("Validated payload", "payload_index", payload.Number, "duration", duration)
	metrics.AddExecutionMetric(NewPayloadLatencyMetric, duration)

	return nil
}

// Start starts the fake consensus client.
func (f *SyncingConsensusClient) Start(ctx context.Context, payloads []engine.ExecutableData, metricsCollector metrics.MetricsCollector, firstTestBlock uint64) error {
	f.log.Info("Starting sync benchmark", "num_payloads", len(payloads))
	for i := 0; i < len(payloads); i++ {
		m := metrics.NewBlockMetrics(payloads[i].Number)
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
