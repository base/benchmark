package payload

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"math/rand"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
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

func (t *TransferOnlyPayloadWorker) Setup(ctx context.Context) error {
	// 21000 * numAccounts
	gasCost := new(big.Int).Mul(big.NewInt(22000*params.GWei), big.NewInt(numAccounts))
	// (prefundAmount - gasCost) / numAccounts
	perAccount := new(big.Int).Div(new(big.Int).Sub(t.prefundAmount, gasCost), big.NewInt(numAccounts))

	sendCalls := make([]rpc.BatchElem, 0, numAccounts)

	nonce := uint64(0)

	results := make([]interface{}, numAccounts)
	var lastTxHash common.Hash

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
		lastTxHash = transferTx.Hash()
	}

	// create batches of 50 txs
	batches := make([][]rpc.BatchElem, 0, (numAccounts+49)/50)
	for i := 0; i < numAccounts; i += 50 {
		batches = append(batches, sendCalls[i:min(i+50, len(sendCalls))])
	}

	for _, batch := range batches {
		if len(batch) == 0 {
			continue
		}

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

	time.Sleep(5 * time.Second)

	receipt, err := t.waitForReceipt(ctx, lastTxHash)
	if err != nil {
		return err
	}

	t.log.Info("Last receipt", "status", receipt.Status)

	t.log.Info("Prefunded accounts", "numAccounts", len(t.accountAddresses), "perAccount", perAccount)

	// update account amounts
	for i := 0; i < numAccounts; i++ {
		t.accountBalances[t.accountAddresses[i]] = perAccount
	}

	return nil
}

func (t *TransferOnlyPayloadWorker) waitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	maxTimeout := time.Now().Add(60 * time.Second)
	for {
		receipt, err := t.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			t.log.Error("Failed to get receipt", "err", err)
		}
		if receipt != nil {
			return receipt, nil
		}
		if time.Now().After(maxTimeout) {
			return nil, fmt.Errorf("timed out waiting for receipt")
		}
		time.Sleep(1 * time.Second)
	}
}

func (t *TransferOnlyPayloadWorker) sendTxs(ctx context.Context, gasLimit uint64) error {
	gasUsed := uint64(0)
	sendCalls := make([]rpc.BatchElem, 0, numAccounts)
	acctIdx := 0

	fakeResults := make([]interface{}, 0)

	for gasUsed < gasLimit {

		transferTx, err := t.createTransferTx(t.accounts[acctIdx], t.accountNonces[t.accountAddresses[acctIdx]], t.accountAddresses[(acctIdx+1)%numAccounts], big.NewInt(1))
		if err != nil {
			t.log.Error("Failed to create transfer transaction", "err", err)
			return err
		}

		marshaledTx, err := transferTx.MarshalBinary()
		if err != nil {
			return err
		}

		fakeResults = append(fakeResults, nil)
		result := fakeResults[len(fakeResults)-1]

		sendCalls = append(sendCalls, rpc.BatchElem{
			Method: "eth_sendRawTransaction",
			Args:   []interface{}{hexutil.Encode(marshaledTx)},
			Result: &result,
		})

		gasUsed += 21000
		t.accountNonces[t.accountAddresses[acctIdx]]++
		// 21000 gas per transfer
		acctIdx = (acctIdx + 1) % numAccounts
	}

	t.log.Debug("created transactions", "numTransactions", len(sendCalls))

	batchSize := 1000

	// create batches of 50 txs
	batches := make([][]rpc.BatchElem, 0, (len(sendCalls)+batchSize-1)/batchSize)
	for i := 0; i < len(sendCalls); i += batchSize {
		batches = append(batches, sendCalls[i:min(i+batchSize, len(sendCalls))])
	}

	t.log.Debug("sending batches", "numBatches", len(batches))
	txIdx := 0

	for _, batch := range batches {
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := t.client.Client().BatchCallContext(callCtx, batch); err != nil {
			return err
		}

		for _, call := range batch {
			if call.Error != nil {
				t.log.Debug("Failed to send transaction", "err", call.Error, "result", call.Result)
				// return call.Error
			}
		}

		txIdx += len(batch)
	}

	t.log.Debug("sent transactions", "numTransactions", len(sendCalls))

	return nil
}

func (t *TransferOnlyPayloadWorker) Start(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			err := t.loop(ctx)
			if err != nil {
				return
			}
		}
	}()
	return nil
}

func (t *TransferOnlyPayloadWorker) createTransferTx(fromPriv *ecdsa.PrivateKey, nonce uint64, toAddr common.Address, amount *big.Int) (*types.Transaction, error) {
	txdata := &types.DynamicFeeTx{
		ChainID:   t.chainID,
		Nonce:     nonce,
		To:        &toAddr,
		Gas:       21000,
		GasFeeCap: new(big.Int).Mul(big.NewInt(params.GWei), big.NewInt(1)),
		GasTipCap: big.NewInt(1),
		Value:     amount,
	}
	signer := types.NewPragueSigner(new(big.Int).SetUint64(t.chainID.Uint64()))
	tx := types.MustSignNewTx(fromPriv, signer, txdata)

	return tx, nil
}

func (t *TransferOnlyPayloadWorker) loop(ctx context.Context) error {
	if err := t.sendTxs(ctx, 21000*10000); err != nil {
		t.log.Error("Failed to send transactions", "err", err)
		return err
	}
	return nil
}
