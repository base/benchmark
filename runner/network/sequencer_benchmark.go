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
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

type sequencerBenchmark struct {
	log                log.Logger
	sequencerClient    types.ExecutionClient
	config             benchtypes.TestConfig
	l1Chain            *l1Chain
	transactionPayload payload.Definition
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
		flashblockCollector = newFlashblockCollector()
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
			BlockTime:     params.BlockTime,
			GasLimit:      params.GasLimit,
			GasLimitSetup: 1e9, // 1G gas
			ParallelTxBatches: nb.config.Config.ParallelTxBatches(),
		}, headBlockHash, headBlockNumber, l1Chain, nb.config.BatcherAddr())

		payloads := make([]engine.ExecutableData, 0)

	setupLoop:
		for {
			_blockMetrics := metrics.NewBlockMetrics()
			_, err := consensusClient.Propose(benchmarkCtx, _blockMetrics, true)
			if err != nil {
				errChan <- err
				return
			}

			select {
			case <-setupComplete:
				break setupLoop
			case <-chainReady:
			default:
				close(chainReady)
			}

		}

		blockMetrics := metrics.NewBlockMetrics()

		// run for a few blocks
		for i := 0; i < params.NumBlocks; i++ {
			blockMetrics.SetBlockNumber(uint64(i) + 1)
			err := transactionWorker.SendTxs(benchmarkCtx)
			if err != nil {
				nb.log.Warn("failed to send transactions", "err", err)
				errChan <- err
				return
			}

			payload, err := consensusClient.Propose(benchmarkCtx, blockMetrics, false)
			if err != nil {
				errChan <- err
				return
			}

			if payload == nil {
				errChan <- errors.New("received nil payload from consensus client")
				return
			}

			time.Sleep(1000 * time.Millisecond)

			err = metricsCollector.Collect(benchmarkCtx, blockMetrics)
			if err != nil {
				nb.log.Error("Failed to collect metrics", "error", err)
			}
			payloads = append(payloads, *payload)
		}

		err = consensusClient.Stop(benchmarkCtx)
		if err != nil {
			nb.log.Warn("failed to stop consensus client", "err", err)
		}

		payloadResult <- payloads
	}()

	select {
	case err := <-errChan:
		return nil, 0, err
	case payloads := <-payloadResult:
		// Collect flashblocks if available
		var flashblocks []types.FlashblocksPayloadV1
		if flashblockCollector != nil {
			flashblocks = flashblockCollector.GetFlashblocks()
			nb.log.Info("Collected flashblocks", "count", len(flashblocks))
		}

		result := &benchtypes.PayloadResult{
			ExecutablePayloads: payloads,
			Flashblocks:        flashblocks,
		}
		return result, payloads[0].Number - 1, nil
	}
}

// flashblockCollector implements FlashblockListener to collect flashblocks.
type flashblockCollector struct {
	flashblocks []types.FlashblocksPayloadV1
	mu          sync.Mutex
}

// newFlashblockCollector creates a new flashblock collector.
func newFlashblockCollector() *flashblockCollector {
	return &flashblockCollector{
		flashblocks: make([]types.FlashblocksPayloadV1, 0),
	}
}

// OnFlashblock implements FlashblockListener.
func (c *flashblockCollector) OnFlashblock(flashblock types.FlashblocksPayloadV1) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flashblocks = append(c.flashblocks, flashblock)
}

// GetFlashblocks returns all collected flashblocks.
func (c *flashblockCollector) GetFlashblocks() []types.FlashblocksPayloadV1 {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return a copy to avoid race conditions
	result := make([]types.FlashblocksPayloadV1, len(c.flashblocks))
	copy(result, c.flashblocks)
	return result
}
