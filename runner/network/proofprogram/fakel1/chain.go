package fakel1

import (
	"context"
	"fmt"
	"math/big"

	"github.com/base/base-bench/runner/network/blocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
)

type FakeL1Chain struct {
	blocks        []*types.Block
	blockReceipts []types.Receipts
	genesis       *core.Genesis
}

func (f *FakeL1Chain) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	for _, block := range f.blocks {
		if block.Hash() == hash {
			return block, nil
		}
	}
	return nil, fmt.Errorf("block not found")
}
func (f *FakeL1Chain) GetBlockByNumber(number uint64) (*types.Block, error) {
	if number >= uint64(len(f.blocks)) {
		return nil, fmt.Errorf("block not found")
	}
	return f.blocks[number], nil
}

func (f *FakeL1Chain) PrintChain(log log.Logger) {
	log.Info("Printing chain")
	for _, block := range f.blocks {
		log.Info("Block", "number", block.NumberU64(), "hash", block.Hash().Hex())
	}
}

func (f *FakeL1Chain) GetLatestBlock() (*types.Block, error) {
	return f.blocks[len(f.blocks)-1], nil
}

func (f *FakeL1Chain) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	for i, block := range f.blocks {
		if block.Hash() == hash {
			return f.blockReceipts[i], nil
		}
	}
	return nil, fmt.Errorf("receipts not found")
}

func (f *FakeL1Chain) AddBlock(block *types.Block, receipts types.Receipts) {
	f.blocks = append(f.blocks, block)
	f.blockReceipts = append(f.blockReceipts, receipts)
}

func (f *FakeL1Chain) BuildAndMine(txs []*types.Transaction) {
	parent := f.blocks[len(f.blocks)-1]

	block := types.NewBlock(&types.Header{
		ParentHash:       parent.Hash(),
		UncleHash:        parent.UncleHash(),
		Root:             parent.Header().Root,
		Number:           new(big.Int).Add(parent.Number(), big.NewInt(1)),
		TxHash:           types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)),
		GasLimit:         parent.GasLimit(),
		GasUsed:          parent.GasUsed(),
		Time:             parent.Time() + 1,
		Extra:            parent.Extra(),
		MixDigest:        parent.MixDigest(),
		BaseFee:          parent.BaseFee(),
		Difficulty:       parent.Difficulty(),
		Coinbase:         parent.Coinbase(),
		Bloom:            parent.Bloom(),
		ReceiptHash:      parent.ReceiptHash(),
		ExcessBlobGas:    parent.ExcessBlobGas(),
		BlobGasUsed:      parent.BlobGasUsed(),
		ParentBeaconRoot: parent.BeaconRoot(),
		WithdrawalsHash:  &types.EmptyWithdrawalsHash,
	}, &types.Body{
		Transactions: txs,
		Withdrawals:  []*types.Withdrawal{},
	}, []*types.Receipt{}, trie.NewStackTrie(nil), blocks.L1BlockType{})

	f.AddBlock(block, []*types.Receipt{})
}

func NewFakeL1ChainWithGenesis(genesis *core.Genesis) *FakeL1Chain {
	blocks := make([]*types.Block, 0)
	blocks = append(blocks, genesis.ToBlock())

	l1Chain := &FakeL1Chain{
		blocks:        blocks,
		blockReceipts: make([]types.Receipts, len(blocks)),
		genesis:       genesis,
	}

	l1Chain.BuildAndMine([]*types.Transaction{})

	return l1Chain
}
