package payload

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"time"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/config"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type Worker interface {
	Setup(ctx context.Context, params benchmark.Params, prefundedPrivateKey []byte, prefundAmount *big.Int) error
	Start(ctx context.Context, elRPCURL string) error
}

type TransferOnlyPayloadWorker struct {
	accounts         []*ecdsa.PrivateKey
	accountAddresses []common.Address
	accountNonces    map[common.Address]uint64
	accountBalances  map[common.Address]*big.Int

	params  benchmark.Params
	chainID *big.Int
}

const numAccounts = 1000

func (t *TransferOnlyPayloadWorker) generateAccounts() error {
	t.accounts = make([]*ecdsa.PrivateKey, 0, numAccounts)

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

func (t *TransferOnlyPayloadWorker) Setup(ctx context.Context, params benchmark.Params, prefundedPrivateKey []byte, prefundAmount *big.Int) error {
	if err := t.generateAccounts(); err != nil {
		return err
	}

	t.params = params
	t.chainID = params.Genesis(time.Now()).Config.ChainID

	// 21000 * numAccounts
	gasCost := new(big.Int).Mul(big.NewInt(21000), big.NewInt(numAccounts))
	// (prefundAmount - gasCost) / numAccounts
	perAccount := new(big.Int).Div(new(big.Int).Sub(prefundAmount, gasCost), big.NewInt(numAccounts))

	priv, _ := btcec.PrivKeyFromBytes(prefundedPrivateKey)

	// prefund accounts
	for i := 0; i < numAccounts; i++ {
		err := t.transfer(priv.ToECDSA(), t.accountAddresses[i], perAccount)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TransferOnlyPayloadWorker) generateTransfers(gasLimit uint64) []*types.Transaction {
	gasUsed := 0
	transactions := make([]*types.Transaction, 0, numAccounts)
	acctIdx := 0
	for {
		// 21000 gas per transfer

		acctIdx = (acctIdx + 1) % numAccounts
	}
}

func (t *TransferOnlyPayloadWorker) Start(ctx context.Context, config config.BenchmarkMatrix, elRPCURL string, prefundedPrivateKey [32]byte) error {

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				err = t.loop()
			}
		}
	}()
}

func (t *TransferOnlyPayloadWorker) transfer(fromPriv *ecdsa.PrivateKey, nonce uint64, toAddr common.Address, amount uint64) error {
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
	}
	signer := types.NewPragueSigner(new(big.Int).SetUint64(t.chainID.Uint64()))
	tx := types.MustSignNewTx(fromPriv, signer, txdata)

	return nil
}

func (t *TransferOnlyPayloadWorker) loop() {

}
