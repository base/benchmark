package network

import (
	"context"
	"time"

	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network/consensus"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/base/base-bench/runner/network/proofprogram/fakel1"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// ReplaySequencerBenchmark is a sequencer benchmark that replays transactions
// from an external node. It has no setup phase - it directly pulls transactions
// from the source node and builds blocks with them.
type ReplaySequencerBenchmark struct {
	log             log.Logger
	sequencerClient types.ExecutionClient
	config          benchtypes.TestConfig
	l1Chain         *l1Chain

	// sourceRPCURL is the RPC endpoint of the node to fetch transactions from
	sourceRPCURL string

	// startBlock is the first block to replay transactions from
	startBlock uint64
}

// NewReplaySequencerBenchmark creates a new replay sequencer benchmark.
func NewReplaySequencerBenchmark(
	log log.Logger,
	config benchtypes.TestConfig,
	sequencerClient types.ExecutionClient,
	l1Chain *l1Chain,
	sourceRPCURL string,
	startBlock uint64,
) *ReplaySequencerBenchmark {
	return &ReplaySequencerBenchmark{
		log:             log,
		config:          config,
		sequencerClient: sequencerClient,
		l1Chain:         l1Chain,
		sourceRPCURL:    sourceRPCURL,
		startBlock:      startBlock,
	}
}

// Run executes the replay benchmark. It fetches transactions from the source
// node block-by-block and replays them on the benchmark node.
func (rb *ReplaySequencerBenchmark) Run(ctx context.Context, metricsCollector metrics.Collector) ([]engine.ExecutableData, uint64, error) {
	params := rb.config.Params
	sequencerClient := rb.sequencerClient

	// Get head block from the snapshot to determine starting point
	headBlockHeader, err := sequencerClient.Client().HeaderByNumber(ctx, nil)
	if err != nil {
		rb.log.Warn("Failed to get head block header", "error", err)
		return nil, 0, err
	}
	headBlockHash := headBlockHeader.Hash()
	headBlockNumber := headBlockHeader.Number.Uint64()

	// Auto-detect start block from snapshot if not specified
	startBlock := rb.startBlock
	if startBlock == 0 {
		// Start from the next block after the snapshot's head
		startBlock = headBlockNumber + 1
		rb.log.Info("Auto-detected start block from snapshot",
			"snapshot_head", headBlockNumber,
			"start_block", startBlock,
		)
	}

	// Create replay mempool that fetches from source node
	replayMempool, err := mempool.NewReplayMempool(
		rb.log,
		rb.sourceRPCURL,
		startBlock,
		rb.config.Genesis.Config.ChainID,
	)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to create replay mempool")
	}
	defer replayMempool.Close()

	benchmarkCtx, benchmarkCancel := context.WithCancel(ctx)
	defer benchmarkCancel()

	errChan := make(chan error)
	payloadResult := make(chan []engine.ExecutableData)

	var l1Chain fakel1.L1Chain
	if rb.l1Chain != nil {
		l1Chain = rb.l1Chain.chain
	}

	go func() {
		consensusClient := consensus.NewSequencerConsensusClient(
			rb.log,
			sequencerClient.Client(),
			sequencerClient.AuthClient(),
			replayMempool,
			consensus.ConsensusClientOptions{
				BlockTime: params.BlockTime,
				GasLimit:  params.GasLimit,
				// No special setup gas limit needed since we're replaying real txs
				GasLimitSetup: params.GasLimit,
				// Allow tx failures for replay since state may differ from source chain
				AllowTxFailures: true,
			},
			headBlockHash,
			headBlockNumber,
			l1Chain,
			rb.config.BatcherAddr(),
		)

		payloads := make([]engine.ExecutableData, 0)
		blockMetrics := metrics.NewBlockMetrics()

		// Directly run benchmark blocks without setup phase
		for i := 0; i < params.NumBlocks; i++ {
			blockMetrics.SetBlockNumber(uint64(i) + 1)

			// Propose will fetch transactions from the replay mempool
			payload, err := consensusClient.Propose(benchmarkCtx, blockMetrics, false)
			if err != nil {
				errChan <- err
				return
			}

			if payload == nil {
				errChan <- errors.New("received nil payload from consensus client")
				return
			}

			rb.log.Info("Built replay block",
				"block", payload.Number,
				"txs", len(payload.Transactions),
				"gas_used", payload.GasUsed,
			)

			time.Sleep(1000 * time.Millisecond)

			err = metricsCollector.Collect(benchmarkCtx, blockMetrics)
			if err != nil {
				rb.log.Error("Failed to collect metrics", "error", err)
			}
			payloads = append(payloads, *payload)
		}

		err = consensusClient.Stop(benchmarkCtx)
		if err != nil {
			rb.log.Warn("Failed to stop consensus client", "error", err)
		}

		payloadResult <- payloads
	}()

	select {
	case err := <-errChan:
		return nil, 0, err
	case payloads := <-payloadResult:
		// No setup blocks for replay, so lastSetupBlock is the head block number
		return payloads, headBlockNumber, nil
	}
}

