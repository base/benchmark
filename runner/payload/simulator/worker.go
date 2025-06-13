package simulator

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/base/base-bench/runner/network/mempool"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/simulator/abi"
	"github.com/base/base-bench/runner/payload/simulator/simulatorstats"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

const maxAccounts = 2

type Bytecode struct {
	Object string `json:"object"`
}

type Contract struct {
	Bytecode Bytecode `json:"bytecode"`
}

type SimulatorPayloadDefinition = simulatorstats.Stats

type simulatorPayloadWorker struct {
	log log.Logger

	params  benchtypes.RunParams
	chainID *big.Int
	client  *ethclient.Client

	prefundedAccount *ecdsa.PrivateKey
	prefundAmount    *big.Int

	mempool *mempool.StaticWorkloadMempool

	contractAddr common.Address

	payloadParams   SimulatorPayloadDefinition
	actualNumConfig SimulatorPayloadDefinition
	numCalls        uint64
	contractBackend *backendWithTrackedNonce

	transactor     *bind.TransactOpts
	callTransactor *bind.CallOpts

	numCallsPerBlock uint64
}

type backendWithTrackedNonce struct {
	bind.ContractBackend
	trackedAddr common.Address
	nonce       uint64
}

func newBackendWithTrackedNonce(transactor bind.ContractBackend, trackedAddr common.Address) (*backendWithTrackedNonce, error) {
	nonce, err := transactor.PendingNonceAt(context.Background(), trackedAddr)
	if err != nil {
		return nil, err
	}

	return &backendWithTrackedNonce{
		ContractBackend: transactor,
		trackedAddr:     trackedAddr,
		nonce:           nonce,
	}, nil
}

func (t *backendWithTrackedNonce) incrementNonce() {
	t.nonce++
}

func (t *backendWithTrackedNonce) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	if account != t.trackedAddr {
		return t.ContractBackend.PendingNonceAt(ctx, account)
	}

	return t.nonce, nil
}

var _ bind.ContractBackend = &backendWithTrackedNonce{}

func NewSimulatorPayloadWorker(ctx context.Context, log log.Logger, elRPCURL string, params benchtypes.RunParams, prefundedPrivateKey ecdsa.PrivateKey, prefundAmount *big.Int, genesis *core.Genesis, payloadParams interface{}) (worker.Worker, error) {
	mempool := mempool.NewStaticWorkloadMempool(log, genesis.Config.ChainID)

	client, err := ethclient.Dial(elRPCURL)
	if err != nil {
		return nil, err
	}

	chainID := genesis.Config.ChainID

	if payloadParams == nil {
		return nil, errors.New("Simulator payload params are required")
	}

	simulatorParams, ok := payloadParams.(*SimulatorPayloadDefinition)
	if !ok {
		return nil, errors.New("Simulator payload params are not valid")
	}

	contractBackend, err := newBackendWithTrackedNonce(client, crypto.PubkeyToAddress(prefundedPrivateKey.PublicKey))
	if err != nil {
		return nil, err
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(&prefundedPrivateKey, chainID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create transactor")
	}
	transactor.NoSend = true

	callTransactor := &bind.CallOpts{
		From:    crypto.PubkeyToAddress(prefundedPrivateKey.PublicKey),
		Context: context.Background(),
	}

	t := &simulatorPayloadWorker{
		log:              log,
		client:           client,
		mempool:          mempool,
		params:           params,
		chainID:          chainID,
		prefundedAccount: &prefundedPrivateKey,
		prefundAmount:    prefundAmount,
		payloadParams:    *simulatorParams,
		contractBackend:  contractBackend,
		transactor:       transactor,
		callTransactor:   callTransactor,
	}

	return t, nil
}

func (t *simulatorPayloadWorker) Mempool() mempool.FakeMempool {
	return t.mempool
}

func (t *simulatorPayloadWorker) Stop(ctx context.Context) error {
	// TODO: Implement
	return nil
}

func (t *simulatorPayloadWorker) mineAndConfirm(ctx context.Context, txs []*types.Transaction) error {
	t.mempool.AddTransactions(txs)

	receipt, err := t.waitForReceipt(ctx, txs[len(txs)-1].Hash())
	if err != nil {
		return errors.Wrap(err, "failed to wait for receipt")
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("contract deployment failed")
	}

	return nil
}

func (t *simulatorPayloadWorker) deployContract(ctx context.Context) (*abi.Simulator, error) {
	contractAddr, contractDeploymentTx, err := t.createDeployTx(t.prefundedAccount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create contract deployment transaction")
	}
	t.contractBackend.incrementNonce()

	t.log.Debug("Contract address", "address", contractAddr.Hex())
	t.contractAddr = *contractAddr

	simulator, err := abi.NewSimulator(t.contractAddr, t.contractBackend)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create simulator transactor")
	}

	if err := t.mineAndConfirm(ctx, []*types.Transaction{contractDeploymentTx}); err != nil {
		return nil, errors.Wrap(err, "failed to mine and confirm contract deployment")
	}

	return simulator, nil
}

// testForBlocks runs the test over 5 blocks and collects max tx gas usage
func (t *simulatorPayloadWorker) testForBlocks(ctx context.Context, simulator *abi.Simulator) error {
	// estimate storage slot usage
	contractConfig, err := t.payloadParams.Mul(float64(t.params.NumBlocks)).ToConfig()
	if err != nil {
		return errors.Wrap(err, "failed to convert payload params to config")
	}

	storageSlotsNeeded, err := simulator.NumStorageSlotsNeeded(t.callTransactor, *contractConfig)
	if err != nil {
		return errors.Wrap(err, "failed to estimate storage slot usage")
	}

	currentStorageSlots, err := simulator.NumStorageInitialized(t.callTransactor)
	if err != nil {
		return errors.Wrap(err, "failed to get current storage slots")
	}

	sendCalls := make([]*types.Transaction, 0)

	storageChunks := uint64(math.Ceil(float64(storageSlotsNeeded.Int64()-currentStorageSlots.Int64()) / 100))

	log.Info("Initializing test storage chunks", "storageChunks", storageChunks)
	for i := uint64(0); i < storageChunks; i++ {
		storageChunkTx, err := simulator.InitializeStorageChunk(t.transactor)
		if err != nil {
			return errors.Wrap(err, "failed to initialize storage chunk")
		}
		t.contractBackend.incrementNonce()

		sendCalls = append(sendCalls, storageChunkTx)
	}

	if err := t.mineAndConfirm(ctx, sendCalls); err != nil {
		return errors.Wrap(err, "failed to mine and confirm storage chunk initialization")
	}

	contractConfig, err = t.payloadParams.ToConfig()
	if err != nil {
		return errors.Wrap(err, "failed to convert payload params to config")
	}

	log.Info("Estimating gas for test run", "run", contractConfig)

	tx, err := simulator.Run(t.transactor, *contractConfig)
	if err != nil {
		return errors.Wrap(err, "failed to run contract")
	}

	gas := tx.Gas()

	// now, we know how many calls we'll actually make, so calculate the storage slots needed
	t.numCallsPerBlock = calcNumCalls(gas, t.params.GasLimit, buffer)

	t.log.Info("Calculated num calls per block", "numCalls", t.numCallsPerBlock, "gas", gas, "gasLimit", t.params.GasLimit, "buffer", buffer)

	configForAllBlocks, err := t.payloadParams.Mul(float64(t.numCallsPerBlock) * float64(t.params.NumBlocks) * 1.05).ToConfig()
	if err != nil {
		return errors.Wrap(err, "failed to convert payload params to config")
	}
	t.log.Info("Calculated config for all blocks", "config", configForAllBlocks)

	storageSlotsNeeded, err = simulator.NumStorageSlotsNeeded(t.callTransactor, *configForAllBlocks)
	if err != nil {
		return errors.Wrap(err, "failed to estimate storage slot usage")
	}

	numExistingStorageSlots, err := simulator.NumStorageInitialized(t.callTransactor)
	if err != nil {
		return errors.Wrap(err, "failed to get number of existing storage slots")
	}

	sendCalls = make([]*types.Transaction, 0)
	additionalStorage := uint64(math.Ceil(float64(storageSlotsNeeded.Int64()-numExistingStorageSlots.Int64()) / 100))
	for i := uint64(0); i < additionalStorage; i++ {
		storageChunkTx, err := simulator.InitializeStorageChunk(t.transactor)
		if err != nil {
			return errors.Wrap(err, "failed to initialize storage chunk")
		}
		t.contractBackend.incrementNonce()

		sendCalls = append(sendCalls, storageChunkTx)
	}

	if err := t.mineAndConfirm(ctx, sendCalls); err != nil {
		return errors.Wrap(err, "failed to mine and confirm storage chunk initialization")
	}

	return nil
}

const buffer = 1e6 // 1M gas buffer to start

func calcNumCalls(gasPerTx uint64, gasLimit uint64, buffer uint64) uint64 {
	return (gasLimit - buffer) / gasPerTx
}

func (t *simulatorPayloadWorker) Setup(ctx context.Context) error {
	// check balance > prefundAmount
	balance, err := t.client.BalanceAt(ctx, crypto.PubkeyToAddress(t.prefundedAccount.PublicKey), nil)
	log.Info("Prefunded account balance", "balance", balance.String())
	if err != nil {
		return errors.Wrap(err, "failed to fetch prefunded account balance")
	}

	if balance.Cmp(t.prefundAmount) < 0 {
		return fmt.Errorf("prefunded account balance %s is less than prefund amount %s", balance.String(), t.prefundAmount.String())
	}

	simulator, err := t.deployContract(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to deploy contract")
	}

	err = t.testForBlocks(ctx, simulator)
	if err != nil {
		return errors.Wrap(err, "failed to test for blocks")
	}

	return nil
}

func (t *simulatorPayloadWorker) waitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return retry.Do(ctx, 60, retry.Fixed(1*time.Second), func() (*types.Receipt, error) {
		receipt, err := t.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	})
}

func (t *simulatorPayloadWorker) sendTxs(ctx context.Context) error {
	txs := make([]*types.Transaction, 0, maxAccounts)

	actual := t.actualNumConfig
	expected := t.payloadParams.Mul(float64(t.numCalls + 1))

	blockCounts := expected.Sub(&actual).Round()
	simulator, err := abi.NewSimulator(t.contractAddr, t.contractBackend)
	if err != nil {
		return errors.Wrap(err, "failed to create simulator transactor")
	}

	for i := uint64(0); i < t.numCallsPerBlock; i++ {

		numInitialized, err := simulator.NumStorageInitialized(t.callTransactor)
		if err != nil {
			return errors.Wrap(err, "failed to get number of storage slots")
		}
		t.log.Info("Number of storage slots initialized", "numInitialized", numInitialized)

		c, err := blockCounts.ToConfig()
		if err != nil {
			return errors.Wrap(err, "failed to convert payload params to config")
		}

		numNeeded, err := simulator.NumStorageSlotsNeeded(t.callTransactor, *c)
		if err != nil {
			return errors.Wrap(err, "failed to get number of storage slots needed")
		}
		t.log.Info("Number of storage slots needed", "numStorageSlotsNeeded", numNeeded)

		transferTx, err := t.createCallTx(t.transactor, t.prefundedAccount, *blockCounts)
		if err != nil {
			t.log.Error("Failed to create transfer transaction", "err", err)
			return err
		}

		t.contractBackend.incrementNonce()

		txs = append(txs, transferTx)

		t.actualNumConfig = *t.actualNumConfig.Add(blockCounts)
		t.numCalls++
	}

	t.mempool.AddTransactions(txs)
	return nil
}

func (t *simulatorPayloadWorker) createCallTx(transactor *bind.TransactOpts, fromPriv *ecdsa.PrivateKey, config SimulatorPayloadDefinition) (*types.Transaction, error) {
	simulator, err := abi.NewSimulator(t.contractAddr, t.contractBackend)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create simulator transactor")
	}

	contractConfig, err := t.payloadParams.ToConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert payload params to config")
	}

	return simulator.Run(transactor, *contractConfig)
}

func (t *simulatorPayloadWorker) createDeployTx(fromPriv *ecdsa.PrivateKey) (*common.Address, *types.Transaction, error) {

	transactor, err := bind.NewKeyedTransactorWithChainID(fromPriv, t.chainID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create transactor")
	}
	transactor.NoSend = true
	transactor.GasLimit = t.params.GasLimit / 2
	transactor.Value = new(big.Int).Div(t.prefundAmount, big.NewInt(2))

	deployAddr, deployTx, _, err := abi.DeploySimulator(transactor, t.contractBackend)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to deploy simulator")
	}

	return &deployAddr, deployTx, nil
}

func (t *simulatorPayloadWorker) SendTxs(ctx context.Context) error {
	if err := t.sendTxs(ctx); err != nil {
		t.log.Error("Failed to send transactions", "err", err)
		return err
	}
	return nil
}
