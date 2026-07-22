package network

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network/consensus"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/base/base-bench/runner/network/proofprogram/fakel1"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload"
	payloadworker "github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

const gracefulWorkerShutdownTimeout = 90 * time.Second

type benchmarkRunController struct {
	maxBlocks  int
	completion payloadworker.CompletionWorker
}

func newBenchmarkRunController(transactionWorker payloadworker.Worker, params benchtypes.RunParams) benchmarkRunController {
	completion, ok := transactionWorker.(payloadworker.CompletionWorker)
	if ok {
		return benchmarkRunController{completion: completion}
	}
	return benchmarkRunController{maxBlocks: params.NumBlocks}
}

// shouldStop reports whether block production should end. Worker completion is
// always a clean stop: a load-test worker that finished its run has produced a
// full metric series even if it exited non-zero (e.g. from expired/abandoned
// submissions under overload). That exit status is a workload outcome, not a
// harness failure, so it is surfaced separately via workerCompletionErr rather
// than aborting here and discarding the collected metrics.
func (c benchmarkRunController) shouldStop(nextBlockIndex uint64) bool {
	if c.completion != nil {
		select {
		case <-c.completion.Done():
			return true
		default:
			return false
		}
	}
	return int(nextBlockIndex) > c.maxBlocks
}

// workerCompletionErr returns the completion worker's exit error, if any. It is
// only meaningful after shouldStop has reported a stop due to completion.
func (c benchmarkRunController) workerCompletionErr() error {
	if c.completion != nil {
		return c.completion.Err()
	}
	return nil
}

func (c benchmarkRunController) usesWorkerCompletion() bool {
	return c.completion != nil
}

type sequencerBenchmark struct {
	log                log.Logger
	sequencerClient    types.ExecutionClient
	config             benchtypes.TestConfig
	l1Chain            *l1Chain
	transactionPayload payload.Definition

	// workloadErr records a non-fatal load-test worker exit error observed when
	// the worker completed its run. It is written by the block-production
	// goroutine before it publishes payloads and read by benchmarkSequencer
	// after Run returns; the payloadResult channel provides the happens-before.
	workloadErr error
}

func newSequencerBenchmark(log log.Logger, config benchtypes.TestConfig, sequencerClient types.ExecutionClient, l1Chain *l1Chain, transactionPayload payload.Definition) *sequencerBenchmark {
	return &sequencerBenchmark{
		log:                log,
		config:             config,
		sequencerClient:    sequencerClient,
		l1Chain:            l1Chain,
		transactionPayload: transactionPayload,
	}
}

func (nb *sequencerBenchmark) fundTestAccount(ctx context.Context, mempool mempool.FakeMempool) error {
	nb.log.Info("Funding test account")
	client := nb.sequencerClient

	addr := crypto.PubkeyToAddress(nb.config.PrefundPrivateKey.PublicKey)

	// fund the test account if needed (check if the account has a balance)
	balance, err := client.Client().BalanceAt(ctx, addr, nil)
	if err != nil {
		nb.log.Warn("failed to get balance", "err", err)
		return err
	}

	blockNumber := uint64(0)
	blockHeader, err := client.Client().HeaderByNumber(ctx, nil)
	if err != nil {
		nb.log.Warn("failed to get block header", "err", err)
		return err
	}
	blockNumber = blockHeader.Number.Uint64()

	random := rand.New(rand.NewSource(int64(blockNumber)))
	randomHash := common.BigToHash(big.NewInt(random.Int63()))

	amount := nb.config.PrefundAmount

	// if balance is already good, return
	if balance.Cmp(&amount) >= 0 {
		return nil
	}

	depositTx := ethTypes.NewTx(
		&ethTypes.DepositTx{
			From:                common.Address{1},
			To:                  &addr,
			SourceHash:          randomHash,
			IsSystemTransaction: false,
			Mint:                &amount,
			Value:               &amount,
			Gas:                 210000,
			Data:                []byte{},
		},
	)

	txHash := depositTx.Hash()

	mempool.AddTransactions([]*ethTypes.Transaction{depositTx})

	// wait for the transaction to be mined
	receipt, err := retry.Do(ctx, 60, retry.Fixed(1*time.Second), func() (*ethTypes.Receipt, error) {
		receipt, err := client.Client().TransactionReceipt(ctx, txHash)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	})
	if receipt == nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	nb.log.Info("Included deposit tx in block", "block", receipt.BlockNumber)
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	if receipt.Status != 1 {
		return fmt.Errorf("transaction failed with status: %d", receipt.Status)
	}

	// ensure balance
	balance, err = client.Client().BalanceAt(ctx, addr, nil)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}
	if balance.Cmp(&amount) < 0 {
		nb.log.Warn("balance is not equal to amount", "balance", balance.String(), "amount", amount.String())
		return errors.New("balance is not equal to amount")
	}
	nb.log.Info("funded test account", "balance", balance, "account", addr.Hex())

	return nil
}

func (nb *sequencerBenchmark) Run(ctx context.Context, metricsCollector metrics.Collector) (*benchtypes.PayloadResult, uint64, error) {
	transactionWorker, err := payload.NewPayloadWorker(ctx, nb.log, &nb.config, nb.sequencerClient, nb.transactionPayload)
	if err != nil {
		return nil, 0, err
	}

	mempool := transactionWorker.Mempool()

	params := nb.config.Params
	sequencerClient := nb.sequencerClient
	defer func() {
		err := transactionWorker.Stop(ctx)
		if err != nil {
			nb.log.Warn("failed to stop payload worker", "err", err)
		}
	}()

	benchmarkCtx, benchmarkCancel := context.WithCancel(ctx)
	defer benchmarkCancel()

	errChan := make(chan error)
	payloadResult := make(chan []engine.ExecutableData)

	setupComplete := make(chan struct{})
	chainReady := make(chan struct{})

	// Check if client supports flashblocks and start collection if available
	var flashblockCollector *flashblockCollector
	flashblocksClient := sequencerClient.FlashblocksClient()
	if flashblocksClient != nil {
		nb.log.Info("Starting flashblocks collection")
		flashblockCollector = newFlashblockCollector(nb.log)
		flashblocksClient.AddListener(flashblockCollector)

		if err := flashblocksClient.Start(benchmarkCtx); err != nil {
			nb.log.Warn("Failed to start flashblocks client", "err", err)
			// Don't fail the benchmark if flashblocks collection fails
		} else {
			defer func() {
				if err := flashblocksClient.Stop(); err != nil {
					nb.log.Warn("Failed to stop flashblocks client", "err", err)
				}
			}()
		}
	}

	go func() {
		// allow one block to pass before sending txs to set the gas limit
		<-chainReady

		err := nb.fundTestAccount(benchmarkCtx, mempool)
		if err != nil {
			nb.log.Warn("failed to fund test account", "err", err)
			errChan <- err
			return
		}

		err = transactionWorker.Setup(benchmarkCtx)
		if err != nil {
			nb.log.Warn("failed to setup payload worker", "err", err)
			errChan <- err
			return
		}
		close(setupComplete)
	}()

	headBlockHeader, err := sequencerClient.Client().HeaderByNumber(ctx, nil)
	if err != nil {
		nb.log.Warn("failed to get head block header", "err", err)
		return nil, 0, err
	}
	headBlockHash := headBlockHeader.Hash()
	headBlockNumber := headBlockHeader.Number.Uint64()

	var l1Chain fakel1.L1Chain
	if nb.l1Chain != nil {
		l1Chain = nb.l1Chain.chain
	}

	go func() {
		consensusClient := consensus.NewSequencerConsensusClient(nb.log, sequencerClient.Client(), sequencerClient.AuthClient(), mempool, consensus.ConsensusClientOptions{
			BlockTime:           params.BlockTime,
			GasLimit:            params.GasLimit,
			GasLimitSetup:       1e9, // 1G gas
			ParallelTxBatches:   nb.config.Config.ParallelTxBatches(),
			ConsensusTimingMode: params.ConsensusTimingMode,
		}, headBlockHash, headBlockNumber, l1Chain, nb.config.BatcherAddr())

		payloads := make([]engine.ExecutableData, 0)

		var lastSetupPayload *engine.ExecutableData
	setupLoop:
		for {
			_blockMetrics := metrics.NewBlockMetrics()
			setupPayload, err := consensusClient.Propose(benchmarkCtx, _blockMetrics, true)
			if err != nil {
				errChan <- err
				return
			}

			lastSetupPayload = setupPayload

			select {
			case <-setupComplete:
				break setupLoop
			case <-chainReady:
			default:
				close(chainReady)
			}

		}

		payloads = append(payloads, *lastSetupPayload)

		pendingTxs := 0
		runController := newBenchmarkRunController(transactionWorker, params)
		if runController.usesWorkerCompletion() {
			nb.log.Info("Running benchmark blocks until payload worker completes")
		}

		blockIndex := uint64(1)
		for {
			if runController.shouldStop(blockIndex) {
				if runController.usesWorkerCompletion() {
					nb.workloadErr = runController.workerCompletionErr()
					nb.log.Info("Payload worker completed", "blocks", blockIndex-1, "workloadErr", nb.workloadErr)
				}
				break
			}

			payload, updatedPendingTxs, err := nb.proposeBlock(
				benchmarkCtx,
				transactionWorker,
				consensusClient,
				metricsCollector,
				blockIndex,
				pendingTxs,
				false,
				true,
			)
			if err != nil {
				errChan <- err
				return
			}
			pendingTxs = updatedPendingTxs
			payloads = append(payloads, *payload)
			blockIndex++
		}

		if !runController.usesWorkerCompletion() {
			if err := nb.settleGracefulWorkerShutdown(benchmarkCtx, transactionWorker, consensusClient, pendingTxs); err != nil {
				nb.log.Warn("settlement blocks failed (non-fatal, metrics already collected)", "err", err)
			}
		}

		if err := consensusClient.Stop(benchmarkCtx); err != nil {
			nb.log.Warn("failed to stop consensus client", "err", err)
		}

		payloadResult <- payloads
	}()

	select {
	case err := <-errChan:
		return nil, 0, err
	case payloads := <-payloadResult:
		// Collect flashblocks if available
		var flashblocks map[uint64][]types.FlashblocksPayloadV1
		if flashblockCollector != nil {
			flashblocks = flashblockCollector.GetFlashblocks()
			nb.log.Info("Collected flashblocks", "count", len(flashblocks))
		}

		result := &benchtypes.PayloadResult{
			ExecutablePayloads: payloads,
			Flashblocks:        flashblocks,
		}
		return result, payloads[0].Number, nil
	}
}

func (nb *sequencerBenchmark) proposeBlock(
	ctx context.Context,
	transactionWorker payloadworker.Worker,
	consensusClient *consensus.SequencerConsensusClient,
	metricsCollector metrics.Collector,
	blockIndex uint64,
	pendingTxs int,
	isSetupPayload bool,
	collectMetrics bool,
) (*engine.ExecutableData, int, error) {
	blockMetrics := metrics.NewBlockMetrics()
	blockMetrics.SetBlockNumber(blockIndex)

	txsSent, err := transactionWorker.SendTxs(ctx, pendingTxs)
	if err != nil {
		nb.log.Warn("failed to send transactions", "err", err)
		return nil, pendingTxs, err
	}

	payload, err := consensusClient.Propose(ctx, blockMetrics, isSetupPayload)
	if err != nil {
		return nil, pendingTxs, err
	}
	if payload == nil {
		return nil, pendingTxs, errors.New("received nil payload from consensus client")
	}

	// Track how many user txs are still pending in the node's mempool.
	// payload.Transactions includes the L1 info deposit tx, so user txs = total - 1.
	userTxsIncluded := len(payload.Transactions) - 1
	if userTxsIncluded < 0 {
		userTxsIncluded = 0
	}
	updatedPendingTxs := pendingTxs + txsSent - userTxsIncluded
	if updatedPendingTxs < 0 {
		updatedPendingTxs = 0
	}

	if !nb.config.Params.UseBaseConsensusTiming() {
		log.Info("Sleeping for block time", "block_time", nb.config.Params.BlockTime)
		time.Sleep(nb.config.Params.BlockTime)
	}

	if collectMetrics {
		if err := metricsCollector.Collect(ctx, blockMetrics); err != nil {
			nb.log.Error("Failed to collect metrics", "error", err)
		}
	}

	return payload, updatedPendingTxs, nil
}

func (nb *sequencerBenchmark) settleGracefulWorkerShutdown(
	ctx context.Context,
	transactionWorker payloadworker.Worker,
	consensusClient *consensus.SequencerConsensusClient,
	pendingTxs int,
) error {
	gracefulWorker, ok := transactionWorker.(payloadworker.GracefulShutdownWorker)
	if !ok {
		return nil
	}

	if err := gracefulWorker.BeginGracefulShutdown(ctx); err != nil {
		return errors.Wrap(err, "failed to begin graceful payload worker shutdown")
	}

	timeout := time.NewTimer(gracefulWorkerShutdownTimeout)
	defer timeout.Stop()

	settlementBlock := 0
	for {
		select {
		case <-gracefulWorker.Done():
			nb.log.Info("Payload worker stopped gracefully", "settlement_blocks", settlementBlock)
			return nil
		case <-timeout.C:
			nb.log.Warn("Timed out waiting for payload worker to stop gracefully", "settlement_blocks", settlementBlock)
			return nil
		default:
		}

		var payload *engine.ExecutableData
		var err error
		payload, pendingTxs, err = nb.proposeBlock(ctx, transactionWorker, consensusClient, nil, uint64(settlementBlock+1), pendingTxs, true, false)
		if err != nil {
			return errors.Wrap(err, "failed to propose settlement block")
		}
		if payload == nil {
			return errors.New("received nil settlement payload from consensus client")
		}

		settlementBlock++
	}
}

// flashblockCollector implements FlashblockListener to collect flashblocks.
type flashblockCollector struct {
	log              log.Logger
	flashblocks      map[uint64][]types.FlashblocksPayloadV1
	currentBaseBlock *uint64
	mu               sync.Mutex
}

// newFlashblockCollector creates a new flashblock collector.
func newFlashblockCollector(log log.Logger) *flashblockCollector {
	return &flashblockCollector{
		flashblocks: make(map[uint64][]types.FlashblocksPayloadV1),
		log:         log,
	}
}

// OnFlashblock implements FlashblockListener.
func (c *flashblockCollector) OnFlashblock(flashblock types.FlashblocksPayloadV1) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if flashblock.Base != nil {
		baseBlock := uint64(flashblock.Base.BlockNumber)
		c.currentBaseBlock = &baseBlock
	} else if c.currentBaseBlock == nil {
		c.log.Warn("received flashblock without base block number")
		return
	}
	c.log.Info("Collected flashblock", "block_number", *c.currentBaseBlock, "index", flashblock.Index, "tx_count", len(flashblock.Diff.Transactions))
	c.flashblocks[*c.currentBaseBlock] = append(c.flashblocks[*c.currentBaseBlock], flashblock)
}

// GetFlashblocks returns all collected flashblocks.
func (c *flashblockCollector) GetFlashblocks() map[uint64][]types.FlashblocksPayloadV1 {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return a copy to avoid race conditions
	result := make(map[uint64][]types.FlashblocksPayloadV1)
	for blockNumber, flashblocks := range c.flashblocks {
		result[blockNumber] = append([]types.FlashblocksPayloadV1{}, flashblocks...)
	}
	return result
}
