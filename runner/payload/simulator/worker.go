package simulator

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"time"

	"github.com/base/base-bench/runner/network/mempool"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/simulator/abi"
	"github.com/base/base-bench/runner/payload/simulator/simulatorstats"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// maxGasPerCall is the maximum gas per call for the payload contract.
const maxGasPerCall = 10000000

// maxStorageSlots is the number of unaccessed storage slots read
const maxStorageSlots = 1e7

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
	numBlocks       uint64
	transactor      *transactorWithTrackedNonce
}

type transactorWithTrackedNonce struct {
	bind.ContractBackend
	trackedAddr common.Address
	nonce       uint64
}

func newTransactorWithTrackedNonce(transactor bind.ContractBackend, trackedAddr common.Address) *transactorWithTrackedNonce {
	return &transactorWithTrackedNonce{
		ContractBackend: transactor,
		trackedAddr:     trackedAddr,
		nonce:           0,
	}
}

func (t *transactorWithTrackedNonce) incrementNonce() {
	t.nonce++
}

func (t *transactorWithTrackedNonce) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	if account != t.trackedAddr {
		return t.ContractBackend.PendingNonceAt(ctx, account)
	}

	return t.nonce, nil
}

var _ bind.ContractBackend = &transactorWithTrackedNonce{}

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

	t := &simulatorPayloadWorker{
		log:              log,
		client:           client,
		mempool:          mempool,
		params:           params,
		chainID:          chainID,
		prefundedAccount: &prefundedPrivateKey,
		prefundAmount:    prefundAmount,
		payloadParams:    *simulatorParams,
		transactor:       newTransactorWithTrackedNonce(client, crypto.PubkeyToAddress(prefundedPrivateKey.PublicKey)),
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

	var lastTxHash common.Hash

	// create contract
	contractFile, err := os.Open("contracts/out/Simulator.sol/Simulator.json")
	if err != nil {
		return errors.Wrap(err, "failed to open contract file")
	}
	defer contractFile.Close()

	var contract Contract
	err = json.NewDecoder(contractFile).Decode(&contract)
	if err != nil {
		return errors.Wrap(err, "failed to decode contract file")
	}

	bytecode, err := hexutil.Decode(contract.Bytecode.Object)
	if err != nil {
		return errors.Wrap(err, "failed to decode contract bytecode")
	}

	numStorageSlotsRequired := uint64((t.payloadParams.StorageLoaded + t.payloadParams.StorageUpdated) * float64(t.params.NumBlocks+2))
	numAccountsRequired := uint64((t.payloadParams.AccountLoaded + t.payloadParams.AccountsUpdated) * float64(t.params.NumBlocks+2))

	storageChunks := uint64(math.Ceil(float64(numStorageSlotsRequired) / 100000))
	accountChunks := uint64(math.Ceil(float64(numAccountsRequired) / 100000))

	contractAddr, contractDeploymentTx, err := t.createDeployTx(t.prefundedAccount, bytecode, numStorageSlotsRequired, numAccountsRequired)
	if err != nil {
		return errors.Wrap(err, "failed to create contract deployment transaction")
	}
	t.transactor.incrementNonce()

	t.log.Debug("Contract address", "address", contractAddr.Hex())
	t.contractAddr = *contractAddr

	t.mempool.AddTransactions([]*types.Transaction{contractDeploymentTx})

	receipt, err := t.waitForReceipt(ctx, contractDeploymentTx.Hash())
	if err != nil {
		return errors.Wrap(err, "failed to wait for receipt")
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("contract deployment failed")
	}

	sendCalls := make([]*types.Transaction, 0)

	transactor, err := bind.NewKeyedTransactorWithChainID(t.prefundedAccount, t.chainID)
	if err != nil {
		return errors.Wrap(err, "failed to create transactor")
	}
	transactor.NoSend = true
	transactor.GasLimit = t.params.GasLimit / 2

	simulator, err := abi.NewSimulator(t.contractAddr, t.transactor)
	if err != nil {
		return errors.Wrap(err, "failed to create simulator transactor")
	}

	for i := uint64(0); i < storageChunks; i++ {
		storageChunkTx, err := simulator.InitializeStorageChunk(transactor, big.NewInt(int64(i)))
		if err != nil {
			return errors.Wrap(err, "failed to initialize storage chunk")
		}
		t.transactor.incrementNonce()
		sendCalls = append(sendCalls, storageChunkTx)
	}

	for i := uint64(0); i < accountChunks; i++ {
		addressChunkTx, err := simulator.InitializeAddressChunk(transactor, big.NewInt(int64(i)))
		if err != nil {
			return errors.Wrap(err, "failed to initialize address chunk")
		}
		t.transactor.incrementNonce()
		sendCalls = append(sendCalls, addressChunkTx)
	}

	lastTxHash = sendCalls[len(sendCalls)-1].Hash()

	t.mempool.AddTransactions(sendCalls)

	receipt, err = t.waitForReceipt(ctx, lastTxHash)
	if err != nil {
		return errors.Wrap(err, "failed to wait for receipt")
	}

	t.log.Debug("Contract deployment receipt", "status", receipt.Status)

	t.log.Debug("Last receipt", "status", receipt.Status)

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
	expected := t.payloadParams.Mul(float64(t.numBlocks + 1))

	blockCounts := expected.Sub(&actual).Round()

	transactor, err := bind.NewKeyedTransactorWithChainID(t.prefundedAccount, t.chainID)
	if err != nil {
		return errors.Wrap(err, "failed to create transactor")
	}
	transactor.NoSend = true
	transactor.GasLimit = t.params.GasLimit / 2

	transferTx, err := t.createCallTx(transactor, t.prefundedAccount, *blockCounts)
	if err != nil {
		t.log.Error("Failed to create transfer transaction", "err", err)
		return err
	}
	t.transactor.incrementNonce()

	txs = append(txs, transferTx)

	t.actualNumConfig = *t.actualNumConfig.Add(blockCounts)
	t.numBlocks++

	t.mempool.AddTransactions(txs)
	return nil
}

func (t *simulatorPayloadWorker) createCallTx(transactor *bind.TransactOpts, fromPriv *ecdsa.PrivateKey, config SimulatorPayloadDefinition) (*types.Transaction, error) {
	simulator, err := abi.NewSimulator(t.contractAddr, t.transactor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create simulator transactor")
	}

	precompiles := make([]abi.PrecompileConfig, 0, len(config.Precompiles))
	for precompileName, numCalls := range config.Precompiles {
		addr, ok := simulatorstats.PrecompileNameToAddress[precompileName]
		if !ok {
			return nil, fmt.Errorf("unknown precompile name: %s", precompileName)
		}

		precompiles = append(precompiles, abi.PrecompileConfig{
			PrecompileAddress: addr,
			NumCalls:          big.NewInt(int64(numCalls)),
		})
	}

	fmt.Printf("config: %+v\n", config)

	return simulator.Run(transactor, abi.SimulatorConfig{
		LoadStorage:    big.NewInt(int64(config.StorageLoaded)),
		UpdateStorage:  big.NewInt(int64(config.StorageUpdated)),
		DeleteStorage:  big.NewInt(int64(config.StorageDeleted)),
		CreateStorage:  big.NewInt(int64(config.StorageCreated)),
		LoadAccounts:   big.NewInt(int64(config.AccountLoaded)),
		UpdateAccounts: big.NewInt(int64(config.AccountsUpdated)),
		DeleteAccounts: big.NewInt(int64(config.AccountDeleted)),
		CreateAccounts: big.NewInt(int64(config.AccountsCreated)),
		Precompiles:    precompiles,
	})
}

func (t *simulatorPayloadWorker) createDeployTx(fromPriv *ecdsa.PrivateKey, contract []byte, numStorageSlotsRequired uint64, numAccountsRequired uint64) (*common.Address, *types.Transaction, error) {

	transactor, err := bind.NewKeyedTransactorWithChainID(fromPriv, t.chainID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create transactor")
	}
	transactor.NoSend = true
	transactor.GasLimit = t.params.GasLimit / 2
	transactor.Value = new(big.Int).Div(t.prefundAmount, big.NewInt(2))

	deployAddr, deployTx, _, err := abi.DeploySimulator(transactor, t.transactor, big.NewInt(int64(numStorageSlotsRequired)), big.NewInt(int64(numAccountsRequired)), big.NewInt(100000), big.NewInt(100000))
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
