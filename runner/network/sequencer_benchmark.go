package network

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network/consensus"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/base/base-bench/runner/network/proofprogram/fakel1"
	"github.com/base/base-bench/runner/payload"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
)

type sequencerBenchmark struct {
	log             log.Logger
	sequencerClient types.ExecutionClient
	config          TestConfig
	l1Chain         *l1Chain
}

func newSequencerBenchmark(log log.Logger, config TestConfig, sequencerClient types.ExecutionClient, l1Chain *l1Chain) *sequencerBenchmark {
	return &sequencerBenchmark{
		log:             log,
		config:          config,
		sequencerClient: sequencerClient,
		l1Chain:         l1Chain,
	}
}

func (nb *sequencerBenchmark) fundTestAccount(ctx context.Context, mempool mempool.FakeMempool, sequencerClient types.ExecutionClient, amount *big.Int) error {
	nb.log.Info("Funding test account")
	client := sequencerClient

	// private key: 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
	addr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")

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

	// if balance is already good, return
	if balance.Cmp(amount) >= 0 {
		return nil
	}

	depositTx := ethTypes.NewTx(
		&ethTypes.DepositTx{
			From:                common.Address{1},
			To:                  &addr,
			SourceHash:          randomHash,
			IsSystemTransaction: false,
			Mint:                amount,
			Value:               amount,
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
	if balance.Cmp(amount) < 0 {
		nb.log.Warn("balance is not equal to amount", "balance", balance, "amount", amount)
		return errors.New("balance is not equal to amount")
	}
	nb.log.Info("funded test account", "balance", balance, "account", addr.Hex())

	return nil
}

func (nb *sequencerBenchmark) Run(ctx context.Context, metricsCollector metrics.MetricsCollector) ([]engine.ExecutableData, uint64, error) {
	amount := new(big.Int).Mul(big.NewInt(1e6), big.NewInt(params.Ether))
	privateKey := common.FromHex("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")

	var mempool mempool.FakeMempool
	var worker payload.Worker
	var err error

	params := nb.config.Params
	config := nb.config.Config
	genesis := nb.config.Genesis

	payloadType := params.TransactionPayload
	sequencerClient := nb.sequencerClient

	switch {
	case payloadType == "tx-fuzz":
		nb.log.Info("Running tx-fuzz payload")
		mempool, worker, err = payload.NewTxFuzzPayloadWorker(
			nb.log, sequencerClient.ClientURL(), params, privateKey, amount, config.TxFuzzBinary())
	case payloadType == "transfer-only":
		mempool, worker, err = payload.NewTransferPayloadWorker(
			ctx, nb.log, sequencerClient.ClientURL(), params, privateKey, amount, &genesis)
	case strings.HasPrefix(string(payloadType), "contract"):
		var payloadConfig payload.ContractPayloadWorkerConfig
		payloadConfig, err = payload.ValidateContractPayload(payloadType, config.ConfigPath())
		if err != nil {
			return nil, 0, err
		}

		mempool, worker, err = payload.NewContractPayloadWorker(
			nb.log, sequencerClient.ClientURL(), params, privateKey, amount, payloadConfig, &genesis)
	default:
		return nil, 0, errors.New("invalid payload type")
	}

	if err != nil {
		return nil, 0, err
	}

	defer func() {
		err := worker.Stop(ctx)
		if err != nil {
			nb.log.Warn("failed to stop payload worker", "err", err)
		}
	}()

	benchmarkCtx, benchmarkCancel := context.WithCancel(ctx)
	defer benchmarkCancel()

	errChan := make(chan error)
	payloadResult := make(chan []engine.ExecutableData)

	setupComplete := make(chan struct{})

	go func() {
		err := nb.fundTestAccount(benchmarkCtx, mempool, sequencerClient, amount)
		if err != nil {
			nb.log.Warn("failed to fund test account", "err", err)
			errChan <- err
			return
		}

		err = worker.Setup(benchmarkCtx)
		if err != nil {
			nb.log.Warn("failed to setup payload worker", "err", err)
			errChan <- err
			return
		}
		close(setupComplete)
	}()

	var lastSetupBlock uint64

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
			BlockTime: params.BlockTime,
			GasLimit:  params.GasLimit,
		}, headBlockHash, headBlockNumber, l1Chain, nb.config.BatcherAddr())

		payloads := make([]engine.ExecutableData, 0)

		// setup blocks
		blockNum := uint64(0)

	setupLoop:
		for {
			_blockMetrics := metrics.NewBlockMetrics(blockNum)
			payload, err := consensusClient.Propose(benchmarkCtx, _blockMetrics)
			if err != nil {
				errChan <- err
				return
			}

			payloads = append(payloads, *payload)
			blockNum = payload.Number
			select {
			case <-setupComplete:
				break setupLoop
			default:
			}
		}

		lastSetupBlock = payloads[len(payloads)-1].Number
		nb.log.Info("Last setup block", "block", lastSetupBlock)

		// run for a few blocks
		for i := 0; i < params.NumBlocks; i++ {
			blockMetrics := metrics.NewBlockMetrics(uint64(i))
			err := worker.SendTxs(benchmarkCtx)
			if err != nil {
				nb.log.Warn("failed to send transactions", "err", err)
				errChan <- err
				return
			}

			payload, err := consensusClient.Propose(benchmarkCtx, blockMetrics)
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
		payloadResult <- payloads
	}()

	select {
	case err := <-errChan:
		return nil, 0, err
	case payloads := <-payloadResult:
		return payloads, lastSetupBlock + 1, nil
	}
}
