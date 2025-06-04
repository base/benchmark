package transferonly

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"math/rand"

	"github.com/base/base-bench/runner/network/mempool"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type TransferOnlyPayloadDefinition struct {
}

type transferOnlyPayloadWorker struct {
	log log.Logger

	privateKeys []*ecdsa.PrivateKey
	addresses   []common.Address
	nextNonce   map[common.Address]uint64
	balance     map[common.Address]*big.Int

	params  benchtypes.RunParams
	chainID *big.Int
	client  *ethclient.Client

	prefundedAccount *ecdsa.PrivateKey
	prefundAmount    *big.Int

	mempool *mempool.StaticWorkloadMempool
}

const numAccounts = 1000

func NewTransferPayloadWorker(ctx context.Context, log log.Logger, elRPCURL string, params benchtypes.RunParams, prefundedPrivateKey ecdsa.PrivateKey, prefundAmount *big.Int, genesis *core.Genesis) (worker.Worker, error) {
	mempool := mempool.NewStaticWorkloadMempool(log)

	client, err := ethclient.Dial(elRPCURL)
	if err != nil {
		return nil, err
	}

	chainID := genesis.Config.ChainID

	t := &transferOnlyPayloadWorker{
		log:              log,
		client:           client,
		mempool:          mempool,
		params:           params,
		chainID:          chainID,
		prefundedAccount: &prefundedPrivateKey,
		prefundAmount:    prefundAmount,
	}

	if err := t.generateAccounts(ctx); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *transferOnlyPayloadWorker) Mempool() mempool.FakeMempool {
	return t.mempool
}

func (t *transferOnlyPayloadWorker) generateAccounts(ctx context.Context) error {
	t.privateKeys = make([]*ecdsa.PrivateKey, 0, numAccounts)
	t.addresses = make([]common.Address, 0, numAccounts)
	t.nextNonce = make(map[common.Address]uint64)
	t.balance = make(map[common.Address]*big.Int)

	src := rand.New(rand.NewSource(100))
	for i := 0; i < numAccounts; i++ {
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
	batchElems := make([]rpc.BatchElem, 0, numAccounts)
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

func (t *transferOnlyPayloadWorker) Stop(ctx context.Context) error {
	// TODO: Implement
	return nil
}

func (t *transferOnlyPayloadWorker) Setup(ctx context.Context) error {
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
	gasCost := new(big.Int).Mul(big.NewInt(21000*params.GWei), big.NewInt(numAccounts))

	// Aim to distribute roughly half of the balance to leave a buffer
	halfBalance := new(big.Int).Div(balance, big.NewInt(2))
	valueToDistribute := new(big.Int).Sub(halfBalance, gasCost)

	// Ensure valueToDistribute is not negative if gasCost is very high or balance is very low
	if valueToDistribute.Sign() < 0 {
		valueToDistribute.SetInt64(0)
	}

	perAccount := new(big.Int).Div(valueToDistribute, big.NewInt(numAccounts))

	// Ensure perAccount is at least 1 wei if we are distributing anything, otherwise it will be 0
	if valueToDistribute.Sign() > 0 && perAccount.Sign() == 0 {
		perAccount.SetInt64(1)
	}

	sendCalls := make([]*types.Transaction, 0, numAccounts)

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

	var lastTxHash common.Hash

	// prefund accounts
	for i := range numAccounts {
		transferTx, err := t.createTransferTx(t.prefundedAccount, nonce, t.addresses[i], perAccount)
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

	t.log.Debug("Prefunded accounts", "numAccounts", len(t.addresses), "perAccount", perAccount)

	// update account amounts
	for i := 0; i < numAccounts; i++ {
		t.balance[t.addresses[i]] = perAccount
	}

	return nil
}

func (t *transferOnlyPayloadWorker) waitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return retry.Do(ctx, 60, retry.Fixed(1*time.Second), func() (*types.Receipt, error) {
		receipt, err := t.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	})
}

func (t *transferOnlyPayloadWorker) sendTxs(ctx context.Context) error {
	gasUsed := uint64(0)
	txs := make([]*types.Transaction, 0, numAccounts)
	acctIdx := 0

	for gasUsed < (t.params.GasLimit - 100_000) {
		transferTx, err := t.createTransferTx(t.privateKeys[acctIdx], t.nextNonce[t.addresses[acctIdx]], t.addresses[(acctIdx+1)%numAccounts], big.NewInt(1))
		if err != nil {
			t.log.Error("Failed to create transfer transaction", "err", err)
			return err
		}

		txs = append(txs, transferTx)

		gasUsed += transferTx.Gas()

		t.nextNonce[t.addresses[acctIdx]]++
		// 21000 gas per transfer
		acctIdx = (acctIdx + 1) % numAccounts
	}

	t.mempool.AddTransactions(txs)
	return nil
}

func (t *transferOnlyPayloadWorker) createTransferTx(fromPriv *ecdsa.PrivateKey, nonce uint64, toAddr common.Address, amount *big.Int) (*types.Transaction, error) {
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

func (t *transferOnlyPayloadWorker) SendTxs(ctx context.Context) error {
	if err := t.sendTxs(ctx); err != nil {
		t.log.Error("Failed to send transactions", "err", err)
		return err
	}
	return nil
}
