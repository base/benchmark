package simulator

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"math/rand"

	"github.com/base/base-bench/runner/network/mempool"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/simulator/simulatorstats"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

// maxGasPerCall is the maximum gas per call for the payload contract.
const maxGasPerCall = 10000000

// maxStorageSlots is the number of unaccessed storage slots read
const maxStorageSlots = 1e7

const maxAccounts = 2

type SimulatorPayloadDefinition = simulatorstats.Stats

func generatePayloadContract(target simulatorstats.Stats) []byte {
	// calculate number of calls to make
	// round(expectedCalls - actualCalls)
	// round((target * (numBlocks + 1)) - (totalPerBlock * numBlocks))
	// numCalls := (target.Copy().Mul(float64(numBlocks + 1))).Sub(totalPerBlock.Copy().Mul(float64(numBlocks)))
	numCalls := target.Round()

	// first read some random account balances and storage slots

	b := []byte{}

	// load call data 0 (storage slot offset)
	b = append(b, byte(vm.PUSH1))
	b = append(b, byte(0))
	b = append(b, byte(vm.CALLDATALOAD))

	for i := 0; i < int(numCalls.StorageLoaded); i++ {
		b = append(b, byte(vm.PUSH1))
		b = append(b, byte(1))
		b = append(b, byte(vm.ADD))
		b = append(b, byte(vm.DUP1))
		b = append(b, byte(vm.SLOAD))
		b = append(b, byte(vm.POP))
	}

	b = append(b, byte(vm.PUSH1))
	b = append(b, byte(1))
	b = append(b, byte(vm.CALLDATALOAD))

	for i := 0; i < int(numCalls.AccountLoaded)-1; i++ {
		b = append(b, byte(vm.PUSH1))
		b = append(b, byte(1))
		b = append(b, byte(vm.ADD))
		b = append(b, byte(vm.DUP1))
		b = append(b, byte(vm.BALANCE))
		b = append(b, byte(vm.POP))
	}

	// pop the account offset
	b = append(b, byte(vm.POP))

	// copy the latest storage offset to use as a counter
	b = append(b, byte(vm.DUP1))

	// store from the latest storage offset backwards
	for i := 0; i < int(numCalls.StorageUpdated); i++ {
		// sub 1 from counter
		b = append(b, byte(vm.PUSH1))
		b = append(b, byte(1))
		b = append(b, byte(vm.SUB))

		// push the key, value to store
		b = append(b, byte(vm.DUP1))
		b = append(b, byte(vm.DUP1))

		// store the value
		b = append(b, byte(vm.SSTORE))
	}

	// pop the counter
	b = append(b, byte(vm.POP))

	for i := 0; i < int(numCalls.StorageCreated); i++ {
		b = append(b, byte(vm.PUSH1))
		b = append(b, byte(1))
		b = append(b, byte(vm.ADD))

		b = append(b, byte(vm.DUP1))
		b = append(b, byte(vm.DUP1))
		b = append(b, byte(vm.SSTORE))
	}

	b = append(b, byte(vm.RETURN))

	deployPrefixSize := byte(16)
	deployPrefix := []byte{
		// Copy input data after this prefix into memory starting at address 0x00
		// CODECOPY arg size
		byte(vm.PUSH1), deployPrefixSize,
		byte(vm.CODESIZE),
		byte(vm.SUB),
		// CODECOPY arg offset
		byte(vm.PUSH1), deployPrefixSize,
		// CODECOPY arg destOffset
		byte(vm.PUSH1), 0x00,
		byte(vm.CODECOPY),

		// Return code from memory
		// RETURN arg size
		byte(vm.PUSH1), deployPrefixSize,
		byte(vm.CODESIZE),
		byte(vm.SUB),
		// RETURN arg offset
		byte(vm.PUSH1), 0x00,
		byte(vm.RETURN),
	}

	b = append(deployPrefix, b...)

	return b
}

type simulatorPayloadWorker struct {
	log log.Logger

	privateKeys  []*ecdsa.PrivateKey
	addresses    []common.Address
	nextNonce    map[common.Address]uint64
	balance      map[common.Address]*big.Int
	prefundNonce uint64

	params        benchtypes.RunParams
	payloadParams SimulatorPayloadDefinition
	chainID       *big.Int
	client        *ethclient.Client

	prefundedAccount *ecdsa.PrivateKey
	prefundAmount    *big.Int

	mempool *mempool.StaticWorkloadMempool

	contractAddr common.Address
}

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
	}

	if err := t.generateAccounts(ctx); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *simulatorPayloadWorker) Mempool() mempool.FakeMempool {
	return t.mempool
}

func (t *simulatorPayloadWorker) generateAccounts(ctx context.Context) error {
	t.privateKeys = make([]*ecdsa.PrivateKey, 0, maxAccounts)
	t.addresses = make([]common.Address, 0, maxAccounts)
	t.nextNonce = make(map[common.Address]uint64)
	t.balance = make(map[common.Address]*big.Int)

	src := rand.New(rand.NewSource(100))
	for i := 0; i < maxAccounts; i++ {
		key, err := ecdsa.GenerateKey(crypto.S256(), src)
		if err != nil {
			return err
		}

		t.privateKeys = append(t.privateKeys, key)
		t.addresses = append(t.addresses, crypto.PubkeyToAddress(key.PublicKey))
		t.nextNonce[crypto.PubkeyToAddress(key.PublicKey)] = 0
		t.balance[crypto.PubkeyToAddress(key.PublicKey)] = big.NewInt(0)
	}

	// fetch nonce and balance for all accounts
	batchElems := make([]rpc.BatchElem, 0, maxAccounts)
	for _, addr := range t.addresses {
		batchElems = append(batchElems, rpc.BatchElem{
			Method: "eth_getTransactionCount",
			Args:   []interface{}{addr, "latest"},
			Result: new(string),
		})
	}

	err := t.client.Client().BatchCallContext(ctx, batchElems)
	if err != nil {
		return errors.Wrap(err, "failed to fetch account nonces")
	}

	for i, elem := range batchElems {
		if elem.Error != nil {
			return errors.Wrapf(elem.Error, "failed to fetch account nonce for %s", t.addresses[i].Hex())
		}
		nonce, err := hexutil.DecodeUint64((*elem.Result.(*string)))
		if err != nil {
			return errors.Wrapf(err, "failed to decode nonce for %s", t.addresses[i].Hex())
		}
		// next nonce
		t.nextNonce[t.addresses[i]] = nonce
	}

	return nil
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

	// 21000 * numAccounts
	gasCost := new(big.Int).Mul(big.NewInt(21000*params.GWei), big.NewInt(maxAccounts))

	// Aim to distribute roughly half of the balance to leave a buffer
	halfBalance := new(big.Int).Div(balance, big.NewInt(2))
	valueToDistribute := new(big.Int).Sub(halfBalance, gasCost)

	// Ensure valueToDistribute is not negative if gasCost is very high or balance is very low
	if valueToDistribute.Sign() < 0 {
		valueToDistribute.SetInt64(0)
	}

	perAccount := new(big.Int).Div(valueToDistribute, big.NewInt(maxAccounts))

	// Ensure perAccount is at least 1 wei if we are distributing anything, otherwise it will be 0
	if valueToDistribute.Sign() > 0 && perAccount.Sign() == 0 {
		perAccount.SetInt64(1)
	}

	sendCalls := make([]*types.Transaction, 0, maxAccounts)

	var nonceHex string
	// fetch nonce for prefunded account
	prefundAddress := crypto.PubkeyToAddress(t.prefundedAccount.PublicKey)
	err = t.client.Client().CallContext(ctx, &nonceHex, "eth_getTransactionCount", prefundAddress.Hex(), "latest")
	if err != nil {
		return errors.Wrap(err, "failed to fetch prefunded account nonce")
	}

	nonce, err := hexutil.DecodeUint64(nonceHex)
	if err != nil {
		return errors.Wrap(err, "failed to decode prefunded account nonce")
	}

	t.prefundNonce = nonce

	var lastTxHash common.Hash

	// prefund accounts
	// for i := 0; i < maxAccounts; i++ {
	// 	transferTx, err := t.createTransferTx(t.prefundedAccount, nonce, t.addresses[i], perAccount)
	// 	if err != nil {
	// 		return errors.Wrap(err, "failed to create transfer transaction")
	// 	}
	// 	nonce++
	// 	sendCalls = append(sendCalls, transferTx)
	// }

	// create contract
	contract := generatePayloadContract(t.payloadParams)
	contractAddr := crypto.CreateAddress(prefundAddress, t.prefundNonce)

	t.log.Debug("Contract address", "address", contractAddr.Hex())
	t.contractAddr = contractAddr

	contractDeploymentTx, err := t.createDeployTx(t.prefundedAccount, t.prefundNonce, contract)
	if err != nil {
		return errors.Wrap(err, "failed to create contract deployment transaction")
	}
	t.prefundNonce++
	sendCalls = append(sendCalls, contractDeploymentTx)
	lastTxHash = contractDeploymentTx.Hash()

	t.mempool.AddTransactions(sendCalls)

	receipt, err := t.waitForReceipt(ctx, lastTxHash)
	if err != nil {
		return errors.Wrap(err, "failed to wait for receipt")
	}

	t.log.Debug("Contract deployment receipt", "status", receipt.Status)

	t.log.Debug("Last receipt", "status", receipt.Status)

	t.log.Debug("Prefunded accounts", "numAccounts", len(t.addresses), "perAccount", perAccount)

	// update account amounts
	for i := 0; i < maxAccounts; i++ {
		t.balance[t.addresses[i]] = perAccount
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
	gasUsed := uint64(0)
	txs := make([]*types.Transaction, 0, maxAccounts)

	storageOffset := uint64(0)
	accountOffset := uint64(0)

	for gasUsed < (t.params.GasLimit - 100_000) {
		transferTx, err := t.createCallTx(t.prefundedAccount, t.prefundNonce, t.contractAddr, storageOffset, accountOffset)
		if err != nil {
			t.log.Error("Failed to create transfer transaction", "err", err)
			return err
		}

		calls := t.payloadParams.Copy().Round()

		storageOffset += uint64(calls.StorageLoaded) + uint64(calls.StorageCreated)
		accountOffset += uint64(calls.AccountsUpdated) + uint64(calls.AccountsCreated)

		txs = append(txs, transferTx)

		gasUsed += transferTx.Gas()

		t.prefundNonce++
	}

	t.mempool.AddTransactions(txs)
	return nil
}

func (t *simulatorPayloadWorker) createTransferTx(fromPriv *ecdsa.PrivateKey, nonce uint64, toAddr common.Address, amount *big.Int) (*types.Transaction, error) {
	txdata := &types.DynamicFeeTx{
		ChainID:   t.chainID,
		Nonce:     nonce,
		To:        &toAddr,
		Gas:       21000,
		GasFeeCap: new(big.Int).Mul(big.NewInt(params.GWei), big.NewInt(1)),
		GasTipCap: big.NewInt(2),
		Value:     amount,
	}
	signer := types.NewPragueSigner(new(big.Int).SetUint64(t.chainID.Uint64()))
	tx := types.MustSignNewTx(fromPriv, signer, txdata)

	return tx, nil
}

func (t *simulatorPayloadWorker) createCallTx(fromPriv *ecdsa.PrivateKey, nonce uint64, toAddr common.Address, currStorageOffset uint64, currAccountOffset uint64) (*types.Transaction, error) {
	currStorageOffsetBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(currStorageOffsetBytes, currStorageOffset)
	currAccountOffsetBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(currAccountOffsetBytes, currAccountOffset)

	txdata := &types.DynamicFeeTx{
		ChainID:   t.chainID,
		Nonce:     nonce,
		To:        &toAddr,
		Gas:       1e6,
		GasFeeCap: new(big.Int).Mul(big.NewInt(params.GWei), big.NewInt(1)),
		GasTipCap: big.NewInt(2),
		Data:      append(currStorageOffsetBytes, currAccountOffsetBytes...),
	}

	signer := types.NewPragueSigner(new(big.Int).SetUint64(t.chainID.Uint64()))
	tx := types.MustSignNewTx(fromPriv, signer, txdata)

	return tx, nil
}

func (t *simulatorPayloadWorker) createDeployTx(fromPriv *ecdsa.PrivateKey, nonce uint64, contract []byte) (*types.Transaction, error) {
	txdata := &types.DynamicFeeTx{
		ChainID:   t.chainID,
		Nonce:     nonce,
		To:        nil,
		Gas:       1e6,
		GasFeeCap: new(big.Int).Mul(big.NewInt(params.GWei), big.NewInt(1)),
		GasTipCap: big.NewInt(2),
		Data:      contract,
	}
	signer := types.NewPragueSigner(new(big.Int).SetUint64(t.chainID.Uint64()))
	tx := types.MustSignNewTx(fromPriv, signer, txdata)

	return tx, nil
}

func (t *simulatorPayloadWorker) SendTxs(ctx context.Context) error {
	if err := t.sendTxs(ctx); err != nil {
		t.log.Error("Failed to send transactions", "err", err)
		return err
	}
	return nil
}
