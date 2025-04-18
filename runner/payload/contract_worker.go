package payload

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const GET_RESULT_SELECTOR = "de292789"

type ContractPayloadWorkerConfig struct {
	Bytecode          []byte
	FunctionSignature string
	Input1            string
	Calldata          []byte
	CallsPerBlock     int
}

type ContractPayloadWorker struct {
	ContractPayloadWorkerConfig

	log log.Logger

	contractAddress common.Address

	params  benchmark.Params
	chainID *big.Int
	client  *ethclient.Client

	prefundedAccount *ecdsa.PrivateKey
	prefundAmount    *big.Int

	mempool *mempool.StaticWorkloadMempool
	nonce   uint64
}

func NewContractPayloadWorker(log log.Logger, elRPCURL string, params benchmark.Params, prefundedPrivateKey []byte, prefundAmount *big.Int, config ContractPayloadWorkerConfig) (mempool.FakeMempool, Worker, error) {
	mempool := mempool.NewStaticWorkloadMempool(log)

	client, err := ethclient.Dial(elRPCURL)
	if err != nil {
		return nil, nil, err
	}

	chainID := params.Genesis(time.Now()).Config.ChainID
	priv, _ := btcec.PrivKeyFromBytes(prefundedPrivateKey)

	t := &ContractPayloadWorker{
		log:                         log,
		client:                      client,
		mempool:                     mempool,
		params:                      params,
		chainID:                     chainID,
		prefundedAccount:            priv.ToECDSA(),
		prefundAmount:               prefundAmount,
		ContractPayloadWorkerConfig: config,
	}

	return mempool, t, nil
}

func (t *ContractPayloadWorker) Stop(ctx context.Context) error {
	// TODO: Implement
	return nil
}

func (t *ContractPayloadWorker) deployContract(ctx context.Context) error {
	address := crypto.PubkeyToAddress(t.prefundedAccount.PublicKey)
	nonce := t.mempool.GetTransactionCount(address)
	t.nonce = nonce

	var gasLimit uint64 = 2000000

	gasPrice, err := t.client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get suggested gas price: %w", err)
	}

	if gasPrice.Cmp(big.NewInt(0)) == 0 {
		gasPrice = big.NewInt(1000000000)
	}

	amount := big.NewInt(0)

	tx_unsigned := types.NewContractCreation(nonce, amount, gasLimit, gasPrice, t.Bytecode)

	signer := types.LatestSignerForChainID(t.chainID)

	tx, err := types.SignTx(tx_unsigned, signer, t.prefundedAccount)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

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

func (t *ContractPayloadWorker) Setup(ctx context.Context) error {
	return t.deployContract(ctx)
}

func (t *ContractPayloadWorker) waitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return retry.Do(ctx, 60, retry.Fixed(1*time.Second), func() (*types.Receipt, error) {
		receipt, err := t.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	})
}

func (t *ContractPayloadWorker) getContractState(ctx context.Context) error {
	debugSelector := common.FromHex(GET_RESULT_SELECTOR)

	var result []byte
	msg := ethereum.CallMsg{
		To:   &t.contractAddress,
		Data: debugSelector,
	}

	result, err := t.client.CallContract(ctx, msg, nil)
	if err != nil {
		return fmt.Errorf("failed to call retrieve: %w", err)
	}

	debugValue := new(big.Int).SetBytes(result)
	t.log.Info("Debug", "value", debugValue.String())

	return nil
}

func (t *ContractPayloadWorker) sendContractTx(ctx context.Context) error {
	contractAddress := t.contractAddress

	privateKey := t.prefundedAccount
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	nonce, err := t.client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	gasLimit := new(big.Int).Mul(big.NewInt(int64(t.params.GasLimit)), big.NewInt(95))
	gasLimit = gasLimit.Div(gasLimit, big.NewInt(int64(t.CallsPerBlock)))
	gasLimit = gasLimit.Div(gasLimit, big.NewInt(100))

	funcSelector := crypto.Keccak256([]byte(t.FunctionSignature))[:4]

	value := new(big.Int)
	value, success := value.SetString(t.Input1, 10)

	if !success {
		return fmt.Errorf("failed to parse input1 as big.Int: %s", t.Input1)
	}

	bytesData := t.Calldata

	uint256Type, _ := abi.NewType("uint256", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)

	arguments := abi.Arguments{
		{
			Type: uint256Type,
		},
		{
			Type: bytesType,
		},
	}
	packedArgs, _ := arguments.Pack(value, bytesData)
	data := append(funcSelector, packedArgs...)

	gasTipCap := big.NewInt(1)
	baseFee := big.NewInt(1e9)

	txdata := &types.DynamicFeeTx{
		Nonce:     nonce,
		Gas:       gasLimit.Uint64(),
		To:        &contractAddress,
		Value:     big.NewInt(0),
		Data:      data,
		GasFeeCap: baseFee,
		GasTipCap: gasTipCap,
		ChainID:   t.chainID,
	}

	signer := types.NewPragueSigner(new(big.Int).SetUint64(t.chainID.Uint64()))
	tx := types.MustSignNewTx(privateKey, signer, txdata)

	err = t.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}
	t.log.Info("Transaction sent", "tx", tx.Hash().Hex())

	return nil
}

func (t *ContractPayloadWorker) SendTxs(ctx context.Context) error {
	err := t.getContractState(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < t.CallsPerBlock; i++ {
		err = t.sendContractTx(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
