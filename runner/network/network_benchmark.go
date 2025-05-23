package network

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/clients"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/ethereum-optimism/optimism/op-service/retry"

	"github.com/base/base-bench/runner/logger"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network/consensus"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/base/base-bench/runner/payload"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
)

const (
	ExecutionLayerLogFileName = "el.log"
)

// NetworkBenchmark handles the lifecycle for a single benchmark run.
type NetworkBenchmark struct {
	log    log.Logger
	params benchmark.Params

	sequencerOptions *config.InternalClientOptions
	validatorOptions *config.InternalClientOptions

	collectedSequencerMetrics *benchmark.SequencerKeyMetrics
	collectedValidatorMetrics *benchmark.ValidatorKeyMetrics

	genesis *core.Genesis
	config  config.Config
}

// NewNetworkBenchmark creates a new network benchmark and initializes the payload worker and consensus client.
func NewNetworkBenchmark(log log.Logger, benchParams benchmark.Params, sequencerOptions *config.InternalClientOptions, validatorOptions *config.InternalClientOptions, genesis *core.Genesis, config config.Config) (*NetworkBenchmark, error) {
	return &NetworkBenchmark{
		log:              log,
		sequencerOptions: sequencerOptions,
		validatorOptions: validatorOptions,
		genesis:          genesis,
		params:           benchParams,
		config:           config,
	}, nil
}

func (nb *NetworkBenchmark) setupNode(ctx context.Context, l log.Logger, params benchmark.Params, options *config.InternalClientOptions) (types.ExecutionClient, error) {
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

func (nb *NetworkBenchmark) Run(ctx context.Context) error {
	payloads, firstTestBlock, err := nb.benchmarkSequencer(ctx)
	if err != nil {
		return err
	}
	return nb.benchmarkValidator(ctx, payloads, firstTestBlock)
}

func (nb *NetworkBenchmark) fundTestAccount(ctx context.Context, mempool mempool.FakeMempool, client types.ExecutionClient, amount *big.Int) error {
	nb.log.Info("Funding test account")

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

func (nb *NetworkBenchmark) benchmarkSequencer(ctx context.Context) ([]engine.ExecutableData, uint64, error) {
	sequencerClient, err := nb.setupNode(ctx, nb.log, nb.params, nb.sequencerOptions)
	if err != nil {
		return nil, 0, err
	}

	defer sequencerClient.Stop()

	// Create metrics collector and writer
	metricsCollector := metrics.NewMetricsCollector(nb.log, sequencerClient.Client(), nb.params.NodeType, sequencerClient.MetricsPort())
	metricsWriter := metrics.NewFileMetricsWriter(nb.sequencerOptions.MetricsPath)

	defer func() {
		sequencerMetrics := metricsCollector.GetMetrics()

		nb.collectedSequencerMetrics = metrics.BlockMetricsToSequencerSummary(sequencerMetrics)

		if err := metricsWriter.Write(sequencerMetrics); err != nil {
			nb.log.Error("Failed to write metrics", "error", err)
		}
	}()

	amount := new(big.Int).Mul(big.NewInt(1e6), big.NewInt(params.Ether))
	privateKey := common.FromHex("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	var mempool mempool.FakeMempool
	var worker payload.Worker

	payloadType := nb.params.TransactionPayload

	switch {
	case payloadType == "tx-fuzz":
		nb.log.Info("Running tx-fuzz payload")
		mempool, worker, err = payload.NewTxFuzzPayloadWorker(
			nb.log, sequencerClient.ClientURL(), nb.params, privateKey, amount, nb.config.TxFuzzBinary())
	case payloadType == "transfer-only":
		mempool, worker, err = payload.NewTransferPayloadWorker(
			ctx, nb.log, sequencerClient.ClientURL(), nb.params, privateKey, amount, nb.genesis)
	case strings.HasPrefix(string(payloadType), "contract"):
		var config payload.ContractPayloadWorkerConfig
		config, err = payload.ValidateContractPayload(payloadType, nb.config.ConfigPath())
		if err != nil {
			return nil, 0, err
		}

		mempool, worker, err = payload.NewContractPayloadWorker(
			nb.log, sequencerClient.ClientURL(), nb.params, privateKey, amount, config, nb.genesis)
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

	go func() {

		consensusClient := consensus.NewSequencerConsensusClient(nb.log, sequencerClient.Client(), sequencerClient.AuthClient(), mempool, consensus.ConsensusClientOptions{
			BlockTime: nb.params.BlockTime,
			GasLimit:  nb.params.GasLimit,
		}, headBlockHash, headBlockNumber)

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
		for i := 0; i < nb.params.NumBlocks; i++ {
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

func (nb *NetworkBenchmark) benchmarkValidator(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64) error {
	validatorClient, err := nb.setupNode(ctx, nb.log, nb.params, nb.validatorOptions)
	if err != nil {
		return err
	}

	defer validatorClient.Stop()

	// Create metrics collector and writer
	metricsCollector := metrics.NewMetricsCollector(nb.log, validatorClient.Client(), nb.params.NodeType, validatorClient.MetricsPort())
	metricsWriter := metrics.NewFileMetricsWriter(nb.validatorOptions.MetricsPath)

	defer func() {
		validatorMetrics := metricsCollector.GetMetrics()

		nb.collectedValidatorMetrics = metrics.BlockMetricsToValidatorSummary(validatorMetrics)

		if err := metricsWriter.Write(validatorMetrics); err != nil {
			nb.log.Error("Failed to write metrics", "error", err)
		}
	}()

	headBlockHeader, err := validatorClient.Client().HeaderByNumber(ctx, nil)
	if err != nil {
		nb.log.Warn("failed to get head block header", "err", err)
	}
	headBlockHash := headBlockHeader.Hash()
	headBlockNumber := headBlockHeader.Number.Uint64()

	consensusClient := consensus.NewSyncingConsensusClient(nb.log, validatorClient.Client(), validatorClient.AuthClient(), consensus.ConsensusClientOptions{
		BlockTime: nb.params.BlockTime,
	}, headBlockHash, headBlockNumber)

	err = consensusClient.Start(ctx, payloads, metricsCollector, firstTestBlock)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		nb.log.Warn("failed to run consensus client", "err", err)
		return err
	}
	return nil
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
