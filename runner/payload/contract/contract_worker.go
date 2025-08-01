package contract

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/network/mempool"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type ContractPayloadDefinition struct {
	ContractBytecode  string `yaml:"contract_bytecode"`
	FunctionSignature string `yaml:"function_signature"`
	GasPerTx          string `yaml:"gas_per_tx"`
	Calldata          string `yaml:"calldata"`
	CallsPerBlock     int    `yaml:"calls_per_block"`
}

type Bytecode struct {
	Object string `json:"object"`
}

type Contract struct {
	Bytecode Bytecode `json:"bytecode"`
}

type contractPayloadWorker struct {
	log log.Logger

	contractAddress common.Address

	runParams benchtypes.RunParams
	params    ContractPayloadDefinition
	chainID   *big.Int
	client    *ethclient.Client

	prefundedAccount *ecdsa.PrivateKey
	prefundAmount    *big.Int

	mempool *mempool.StaticWorkloadMempool
	nonce   uint64

	config   config.Config
	bytecode []byte
}

func NewContractPayloadWorker(log log.Logger, elRPCURL string, runParams benchtypes.RunParams, prefundedPrivateKey ecdsa.PrivateKey, prefundAmount *big.Int, genesis *core.Genesis, config config.Config, params interface{}) (worker.Worker, error) {
	mempool := mempool.NewStaticWorkloadMempool(log, genesis.Config.ChainID)

	client, err := ethclient.Dial(elRPCURL)
	if err != nil {
		return nil, err
	}

	chainID := genesis.Config.ChainID

	if params == nil {
		return nil, fmt.Errorf("transaction payload params are missing, but required for contract payload worker")
	}

	payloadConfig, ok := (params).(*ContractPayloadDefinition)
	if !ok {
		return nil, fmt.Errorf("invalid contract transaction payload: %#v", params)
	}

	bytecodeFile := payloadConfig.ContractBytecode

	bytecodePath := filepath.Join("contracts/out/" + bytecodeFile + ".sol/" + bytecodeFile + ".json")
	data, err := os.ReadFile(bytecodePath)
	if err != nil {
		return nil, errors.New("failed to read bytecode file")
	}

	var c Contract
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, errors.New("failed to unmarshal bytecode file")
	}

	bytecodeHex := c.Bytecode.Object

	bytecode := common.FromHex(string(bytecodeHex))

	t := &contractPayloadWorker{
		log:              log,
		client:           client,
		mempool:          mempool,
		runParams:        runParams,
		chainID:          chainID,
		prefundedAccount: &prefundedPrivateKey,
		prefundAmount:    prefundAmount,
		params:           *payloadConfig,
		config:           config,
		bytecode:         bytecode,
	}

	return t, nil
}

func (t *contractPayloadWorker) Mempool() mempool.FakeMempool {
	return t.mempool
}

func (t *contractPayloadWorker) Stop(ctx context.Context) error {
	// TODO: Implement
	return nil
}

func (t *contractPayloadWorker) deployContract(ctx context.Context) error {
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

	tx_unsigned := types.NewContractCreation(nonce, amount, gasLimit, gasPrice, t.bytecode)

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

func (t *contractPayloadWorker) Setup(ctx context.Context) error {
	return t.deployContract(ctx)
}

func (t *contractPayloadWorker) waitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return retry.Do(ctx, 60, retry.Fixed(1*time.Second), func() (*types.Receipt, error) {
		receipt, err := t.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	})
}

func (t *contractPayloadWorker) debugContract() (*big.Int, error) {
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

func (t *contractPayloadWorker) sendContractTx(ctx context.Context) error {
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

	gasLimit := new(big.Int).Mul(big.NewInt(int64(t.runParams.GasLimit)), big.NewInt(95))
	gasLimit = gasLimit.Div(gasLimit, big.NewInt(int64(t.params.CallsPerBlock)))
	gasLimit = gasLimit.Div(gasLimit, big.NewInt(100))

	funcSelector := crypto.Keccak256([]byte(t.params.FunctionSignature))[:4]

	value := new(big.Int)
	value, success := value.SetString(t.params.GasPerTx, 10)

	if !success {
		return fmt.Errorf("failed to parse gas per tx as big.Int: %s", t.params.GasPerTx)
	}

	bytesHex := t.params.Calldata
	if bytesHex == "" {
		bytesHex = "0x"
	}
	bytesData, err := hexutil.Decode(bytesHex)
	if err != nil {
		return fmt.Errorf("failed to decode calldata: %w", err)
	}

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

func (t *contractPayloadWorker) SendTxs(ctx context.Context) error {
	for i := 0; i < t.params.CallsPerBlock; i++ {
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
