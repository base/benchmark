package payload

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"time"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type Worker interface {
	Setup(ctx context.Context) error
	Start(ctx context.Context) error
}

type NewWorkerFn func(logger log.Logger, elRPCURL string, params benchmark.Params, prefundedPrivateKey []byte, prefundAmount *big.Int) (Worker, error)

type TransferOnlyPayloadWorker struct {
	log log.Logger

	accounts         []*ecdsa.PrivateKey
	accountAddresses []common.Address
	accountNonces    map[common.Address]uint64
	accountBalances  map[common.Address]*big.Int

	params  benchmark.Params
	chainID *big.Int
	client  *ethclient.Client

	prefundedAccount *ecdsa.PrivateKey
	prefundAmount    *big.Int
}

const numAccounts = 1000

func NewTransferPayloadWorker(log log.Logger, elRPCURL string, params benchmark.Params, prefundedPrivateKey []byte, prefundAmount *big.Int) (Worker, error) {
	client, err := ethclient.Dial(elRPCURL)
	if err != nil {
		return nil, err
	}

	chainID := params.Genesis(time.Now()).Config.ChainID
	priv, _ := btcec.PrivKeyFromBytes(prefundedPrivateKey)

	t := &TransferOnlyPayloadWorker{
		log:              log,
		client:           client,
		params:           params,
		chainID:          chainID,
		prefundedAccount: priv.ToECDSA(),
		prefundAmount:    prefundAmount,
	}

	if err := t.generateAccounts(); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *TransferOnlyPayloadWorker) generateAccounts() error {
	t.accounts = make([]*ecdsa.PrivateKey, 0, numAccounts)
	t.accountAddresses = make([]common.Address, 0, numAccounts)
	t.accountNonces = make(map[common.Address]uint64)
	t.accountBalances = make(map[common.Address]*big.Int)

	for i := 0; i < numAccounts; i++ {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
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

func (t *TransferOnlyPayloadWorker) Setup(ctx context.Context) error {
	// 21000 * numAccounts
	gasCost := new(big.Int).Mul(big.NewInt(21000), big.NewInt(numAccounts))
	// (prefundAmount - gasCost) / numAccounts
	perAccount := new(big.Int).Div(new(big.Int).Sub(t.prefundAmount, gasCost), big.NewInt(numAccounts))

	sendCalls := make([]rpc.BatchElem, 0, numAccounts)

	nonce := uint64(0)

	results := make([]interface{}, numAccounts)

	// prefund accounts
	for i := 0; i < numAccounts; i++ {

		transferTx, err := t.createTransferTx(t.prefundedAccount, nonce, t.accountAddresses[i], perAccount)
		if err != nil {
			return err
		}
		nonce++

		marshaledTx, err := transferTx.MarshalBinary()
		if err != nil {
			return err
		}

		sendCalls = append(sendCalls, rpc.BatchElem{
			Method: "eth_sendRawTransaction",
			Args:   []interface{}{hexutil.Encode(marshaledTx)},
			Result: &results[i],
		})
	}

	// create batches of 50 txs
	batches := make([][]rpc.BatchElem, 0, numAccounts/50)
	for i := 0; i < numAccounts; i += 50 {
		batches = append(batches, sendCalls[i:i+50])
	}

	for _, batch := range batches {
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := t.client.Client().BatchCallContext(callCtx, batch); err != nil {
			return err
		}

		for _, call := range sendCalls {
			if call.Error != nil {
				t.log.Debug("Failed to send transaction", "err", call.Error, "result", call.Result)
				return call.Error
			}
		}

		t.log.Info("Sent batch of transactions", "numTransactions", len(batch))
	}

	t.log.Info("Prefunded accounts", "numAccounts", len(t.accountAddresses), "perAccount", perAccount)

	// update account amounts
	for i := 0; i < numAccounts; i++ {
		t.accountBalances[t.accountAddresses[i]] = perAccount
	}

	return nil
}

// func (t *TransferOnlyPayloadWorker) generateTransfers(gasLimit uint64) []*types.Transaction {
// 	gasUsed := 0
// 	transactions := make([]*types.Transaction, 0, numAccounts)
// 	acctIdx := 0
// 	for {
// 		// 21000 gas per transfer
// 		acctIdx = (acctIdx + 1) % numAccounts

// 	}
// }

func (t *TransferOnlyPayloadWorker) Start(ctx context.Context) error {

	// go func() {
	// 	ticker := time.NewTicker(1 * time.Second)
	// 	defer ticker.Stop()

	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		default:
	// 			err = t.loop()
	// 		}
	// 	}
	// }()
	return nil
}

func (t *TransferOnlyPayloadWorker) createTransferTx(fromPriv *ecdsa.PrivateKey, nonce uint64, toAddr common.Address, amount *big.Int) (*types.Transaction, error) {
	// ChainID    *big.Int
	// Nonce      uint64
	// GasTipCap  *big.Int // a.k.a. maxPriorityFeePerGas
	// GasFeeCap  *big.Int // a.k.a. maxFeePerGas
	// Gas        uint64
	// To         *common.Address `rlp:"nil"` // nil means contract creation
	// Value      *big.Int
	// Data       []byte
	// AccessList AccessList
	txdata := &types.DynamicFeeTx{
		ChainID:   t.chainID,
		Nonce:     nonce,
		To:        &toAddr,
		Gas:       21000,
		GasFeeCap: big.NewInt(5000000000),
		GasTipCap: big.NewInt(0),
		Value:     amount,
	}
	signer := types.NewPragueSigner(new(big.Int).SetUint64(t.chainID.Uint64()))
	tx := types.MustSignNewTx(fromPriv, signer, txdata)

	return tx, nil
}

func (t *TransferOnlyPayloadWorker) loop() {

}
