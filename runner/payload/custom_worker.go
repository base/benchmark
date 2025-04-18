package payload

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"math/rand"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type CustomPayloadWorker struct {
	log log.Logger

	accounts         []*ecdsa.PrivateKey
	accountAddresses []common.Address
	accountNonces    map[common.Address]uint64
	accountBalances  map[common.Address]*big.Int

	contractAddress common.Address

	params  benchmark.Params
	chainID *big.Int
	client  *ethclient.Client

	prefundedAccount *ecdsa.PrivateKey
	prefundAmount    *big.Int

	mempool *mempool.StaticWorkloadMempool

	nonce uint64
}

func NewCustomPayloadWorker(log log.Logger, elRPCURL string, params benchmark.Params, prefundedPrivateKey []byte, prefundAmount *big.Int) (mempool.FakeMempool, Worker, error) {
	mempool := mempool.NewStaticWorkloadMempool(log)

	client, err := ethclient.Dial(elRPCURL)
	if err != nil {
		return nil, nil, err
	}

	chainID := params.Genesis(time.Now()).Config.ChainID
	priv, _ := btcec.PrivKeyFromBytes(prefundedPrivateKey)

	t := &CustomPayloadWorker{
		log:              log,
		client:           client,
		mempool:          mempool,
		params:           params,
		chainID:          chainID,
		prefundedAccount: priv.ToECDSA(),
		prefundAmount:    prefundAmount,
	}

	if err := t.generateAccounts(); err != nil {
		return nil, nil, err
	}

	return mempool, t, nil
}

func (t *CustomPayloadWorker) generateAccounts() error {
	t.accounts = make([]*ecdsa.PrivateKey, 0, numAccounts)
	t.accountAddresses = make([]common.Address, 0, numAccounts)
	t.accountNonces = make(map[common.Address]uint64)
	t.accountBalances = make(map[common.Address]*big.Int)

	src := rand.New(rand.NewSource(100))
	for i := 0; i < numAccounts; i++ {
		key, err := ecdsa.GenerateKey(btcec.S256(), src)
		if err != nil {
			return err
		}

		t.accounts = append(t.accounts, key)
		t.accountAddresses = append(t.accountAddresses, crypto.PubkeyToAddress(key.PublicKey))
		t.accountNonces[crypto.PubkeyToAddress(key.PublicKey)] = 0
		t.accountBalances[crypto.PubkeyToAddress(key.PublicKey)] = big.NewInt(0)
	}

	return nil
}

func (t *CustomPayloadWorker) Stop(ctx context.Context) error {
	// TODO: Implement
	return nil
}

// this will be called in the setup function, it just deployes the smart contract using the prefundedAccount
func (t *CustomPayloadWorker) basicSmartContract(ctx context.Context) error {
	address := crypto.PubkeyToAddress(t.prefundedAccount.PublicKey)
	nonce := t.mempool.GetTransactionCount(address)
	t.nonce = nonce

	var gasLimit uint64 = 1000000

	// Get suggested gas price from the network
	gasPrice, err := t.client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get suggested gas price: %w", err)
	}

	var amount *big.Int = big.NewInt(0)

	storeSelector := crypto.Keccak256([]byte("store(uint256)"))[:4]
	retrieveSelector := crypto.Keccak256([]byte("retrieve()"))[:4]

	runtime := []byte{
		// function dispatcher
		byte(vm.CALLDATALOAD),
		byte(vm.PUSH4), storeSelector[0], storeSelector[1], storeSelector[2], storeSelector[3],
		byte(vm.EQ),
		byte(vm.PUSH1), 0x17, // JUMPDEST for store()
		byte(vm.JUMPI),

		byte(vm.CALLDATALOAD),
		byte(vm.PUSH4), retrieveSelector[0], retrieveSelector[1], retrieveSelector[2], retrieveSelector[3],
		byte(vm.EQ),
		byte(vm.PUSH1), 0x1d, // JUMPDEST for retrieve()
		byte(vm.JUMPI),

		byte(vm.STOP),

		// store():
		byte(vm.JUMPDEST), // 0x17
		byte(vm.CALLDATALOAD),
		byte(vm.PUSH1), 0x00,
		byte(vm.SSTORE),
		byte(vm.STOP),

		// retrieve():
		byte(vm.JUMPDEST), // 0x1d
		byte(vm.PUSH1), 0x00,
		byte(vm.SLOAD),
		byte(vm.PUSH1), 0x00,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 0x20,
		byte(vm.PUSH1), 0x00,

		byte(vm.RETURN),
	}

	constructor := []byte{
		byte(vm.PUSH1), byte(len(runtime)),
		byte(vm.PUSH1), byte(0x0a), // hardcode offset = 10 (length of constructor)
		byte(vm.PUSH1), 0x00,
		byte(vm.CODECOPY),
		byte(vm.PUSH1), byte(len(runtime)),
		byte(vm.PUSH1), 0x00,
		byte(vm.RETURN),
	}

	data := append(constructor, runtime...)

	tx_unsigned := types.NewContractCreation(nonce, amount, gasLimit, gasPrice, data)

	// Use the appropriate signer for your network
	signer := types.LatestSignerForChainID(t.chainID)

	tx, err := types.SignTx(tx_unsigned, signer, t.prefundedAccount)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send the transaction to the network
	err = t.client.SendTransaction(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	t.contractAddress = crypto.CreateAddress(address, nonce)
	t.log.Info("Contract address", "address", t.contractAddress)

	receipt, err := t.waitForReceipt(ctx, tx.Hash())
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("contract deployment failed with status: %d", receipt.Status)
	}

	t.nonce++

	t.log.Info("Contract deployed successfully", "receipt", receipt)
	return nil
}

func (t *CustomPayloadWorker) Setup(ctx context.Context) error {
	return t.basicSmartContract(ctx)
}

func (t *CustomPayloadWorker) waitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return retry.Do(ctx, 60, retry.Fixed(1*time.Second), func() (*types.Receipt, error) {
		receipt, err := t.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	})
}

func (t *CustomPayloadWorker) sendStoreTx(ctx context.Context) error {
	address := crypto.PubkeyToAddress(t.prefundedAccount.PublicKey)

	gasPrice, err := t.client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get suggested gas price: %w", err)
	}

	// Get gas limit estimate
	msg := ethereum.CallMsg{
		From:  address,
		To:    &t.contractAddress,
		Value: big.NewInt(0),
		Data:  append(crypto.Keccak256([]byte("store(uint256)"))[:4], common.LeftPadBytes(big.NewInt(0).Bytes(), 32)...),
	}
	gasLimit, err := t.client.EstimateGas(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to estimate gas: %w", err)
	}

	// Add some buffer to the gas limit
	gasLimit = gasLimit * 2

	contractAddress := t.contractAddress
	storeSelector := crypto.Keccak256([]byte("store(uint256)"))[:4]
	value := new(big.Int).SetUint64(0)
	encodedValue := common.LeftPadBytes(value.Bytes(), 32)
	data := append(storeSelector, encodedValue...)

	tx_unsigned := types.NewTransaction(t.nonce, contractAddress, nil, gasLimit, gasPrice, data)

	signer := types.LatestSignerForChainID(t.chainID)
	tx, err := types.SignTx(tx_unsigned, signer, t.prefundedAccount)
	if err != nil {
		return fmt.Errorf("failed to sign store transaction: %w", err)
	}

	// Send the transaction to the network
	err = t.client.SendTransaction(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to send store transaction: %w", err)
	}

	// Increment nonce after successful broadcast
	t.nonce++

	return nil
}

// func (t *CustomPayloadWorker) sendRetrieveTx(ctx context.Context) error {
// 	return nil
// }

func (t *CustomPayloadWorker) SendTxs(ctx context.Context) error {
	err := t.sendStoreTx(ctx)
	if err != nil {
		return err
	}

	return nil
}
