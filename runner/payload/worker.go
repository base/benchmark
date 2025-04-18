package payload

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	"math/rand"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
)

// Note: Payload workers are responsible keeping track of gas in a block and sending transactions to the mempool.
type Worker interface {
	Setup(ctx context.Context) error
	SendTxs(ctx context.Context) error
	Stop(ctx context.Context) error
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

	mempool *mempool.StaticWorkloadMempool
}

const numAccounts = 10000

func NewTransferPayloadWorker(log log.Logger, elRPCURL string, params benchmark.Params, prefundedPrivateKey []byte, prefundAmount *big.Int) (mempool.FakeMempool, Worker, error) {
	mempool := mempool.NewStaticWorkloadMempool(log)

	client, err := ethclient.Dial(elRPCURL)
	if err != nil {
		return nil, nil, err
	}

	chainID := params.Genesis(time.Now()).Config.ChainID
	priv, _ := btcec.PrivKeyFromBytes(prefundedPrivateKey)

	t := &TransferOnlyPayloadWorker{
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

func (t *TransferOnlyPayloadWorker) Stop(ctx context.Context) error {
	// TODO: Implement
	return nil
}

func (t *TransferOnlyPayloadWorker) Setup(ctx context.Context) error {
	// 21000 * numAccounts
	gasCost := new(big.Int).Mul(big.NewInt(22000*params.GWei), big.NewInt(numAccounts))
	// (prefundAmount - gasCost) / numAccounts
	perAccount := new(big.Int).Div(new(big.Int).Sub(t.prefundAmount, gasCost), big.NewInt(numAccounts))

	sendCalls := make([]*types.Transaction, 0, numAccounts)

	nonce := uint64(0)

	var lastTxHash common.Hash

	// prefund accounts
	for i := 0; i < numAccounts; i++ {

		transferTx, err := t.createTransferTx(t.prefundedAccount, nonce, t.accountAddresses[i], perAccount)
		if err != nil {
			return errors.Wrap(err, "failed to create transfer transaction")
		}
		nonce++
		sendCalls = append(sendCalls, transferTx)
		lastTxHash = transferTx.Hash()
	}

	t.mempool.AddTransactions(sendCalls)

	receipt, err := t.waitForReceipt(ctx, lastTxHash)
	if err != nil {
		return errors.Wrap(err, "failed to wait for receipt")
	}

	t.log.Debug("Last receipt", "status", receipt.Status)

	t.log.Debug("Prefunded accounts", "numAccounts", len(t.accountAddresses), "perAccount", perAccount)

	// update account amounts
	for i := 0; i < numAccounts; i++ {
		t.accountBalances[t.accountAddresses[i]] = perAccount
	}

	return nil
}

func (t *TransferOnlyPayloadWorker) waitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return retry.Do(ctx, 60, retry.Fixed(1*time.Second), func() (*types.Receipt, error) {
		receipt, err := t.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	})
}

func (t *TransferOnlyPayloadWorker) sendTxs(ctx context.Context, gasLimit uint64) error {
	gasUsed := uint64(0)
	txs := make([]*types.Transaction, 0, numAccounts)
	acctIdx := 0

	for gasUsed < gasLimit {
		transferTx, err := t.createTransferTx(t.accounts[acctIdx], t.accountNonces[t.accountAddresses[acctIdx]], t.accountAddresses[(acctIdx+1)%numAccounts], big.NewInt(1))
		if err != nil {
			t.log.Error("Failed to create transfer transaction", "err", err)
			return err
		}

		txs = append(txs, transferTx)

		gasUsed += 21000
		t.accountNonces[t.accountAddresses[acctIdx]]++
		// 21000 gas per transfer
		acctIdx = (acctIdx + 1) % numAccounts
	}

	t.mempool.AddTransactions(txs)
	return nil
}

func (t *TransferOnlyPayloadWorker) createTransferTx(fromPriv *ecdsa.PrivateKey, nonce uint64, toAddr common.Address, amount *big.Int) (*types.Transaction, error) {
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

func (t *TransferOnlyPayloadWorker) SendTxs(ctx context.Context) error {
	if err := t.sendTxs(ctx, 21000*10000); err != nil {
		t.log.Error("Failed to send transactions", "err", err)
		return err
	}
	return nil
}
