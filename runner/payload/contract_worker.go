package payload

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

type ContractPayloadWorkerConfig struct {
	Bytecode          []byte
	FunctionSignature string
	Input1            string
	Calldata          []byte
	CallsPerBlock     int
}

func ValidateContractPayload(payloadType benchmark.TransactionPayload, configDir string) (ContractPayloadWorkerConfig, error) {
	selectors := strings.Split(string(payloadType), ":")

	if len(selectors) != 6 {
		return ContractPayloadWorkerConfig{}, errors.New("invalid contract payload type")
	}

	var callsPerBlock int
	callsPerBlock, err := strconv.Atoi(selectors[1])
	if err != nil {
		return ContractPayloadWorkerConfig{}, errors.New("invalid calls per block")
	}

	functionSignature := selectors[2]
	input1 := selectors[3]

	calldata := common.FromHex(selectors[4])

	bytecodeFile := selectors[5]

	dir := filepath.Dir(configDir)

	bytecodePath := filepath.Join(dir, bytecodeFile)
	log.Info("loading bytecode from", "path", bytecodePath)

	bytecodeHex, err := os.ReadFile(bytecodePath)
	if err != nil {
		return ContractPayloadWorkerConfig{}, errors.New("failed to read bytecode file")
	}
	bytecode := common.FromHex(string(bytecodeHex))

	config := ContractPayloadWorkerConfig{
		Bytecode:          bytecode,
		FunctionSignature: functionSignature,
		Input1:            input1,
		Calldata:          calldata,
		CallsPerBlock:     callsPerBlock,
	}
	return config, nil
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

func NewContractPayloadWorker(log log.Logger, elRPCURL string, params benchmark.Params, prefundedPrivateKey []byte, prefundAmount *big.Int, config ContractPayloadWorkerConfig, genesis *core.Genesis) (mempool.FakeMempool, Worker, error) {
	mempool := mempool.NewStaticWorkloadMempool(log)

	client, err := ethclient.Dial(elRPCURL)
	if err != nil {
		return nil, nil, err
	}

	chainID := genesis.Config.ChainID
	priv, err := crypto.ToECDSA(prefundedPrivateKey)
	if err != nil {
		return nil, nil, err
	}

	t := &ContractPayloadWorker{
		log:                         log,
		client:                      client,
		mempool:                     mempool,
		params:                      params,
		chainID:                     chainID,
		prefundedAccount:            priv,
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

func (t *ContractPayloadWorker) debugContract() (*big.Int, error) {
	contractAddress := t.contractAddress

	fromAddress := crypto.PubkeyToAddress(t.prefundedAccount.PublicKey)

	functionSignature := "getResult()"
	funcSelector := crypto.Keccak256([]byte(functionSignature))[:4]

	msg := ethereum.CallMsg{
		From: fromAddress,
		To:   &contractAddress,
		Data: funcSelector,
	}

	ctx := context.Background()
	res, err := t.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	result := new(big.Int).SetBytes(res)
	return result, nil
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

	t.mempool.AddTransactions([]*types.Transaction{tx})

	return nil
}

func (t *ContractPayloadWorker) SendTxs(ctx context.Context) error {
	for i := 0; i < t.CallsPerBlock; i++ {
		err := t.sendContractTx(ctx)

		if err != nil {
			t.log.Error("Failed to send transaction", "error", err)
			return err
		}

		debugResult, err := t.debugContract()
		if err == nil {
			t.log.Debug("getResult()", "result", debugResult)
		}
	}

	return nil
}
