package fakel1

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type FakeL1Chain struct {
	blocks        []*types.Block
	blockReceipts []types.Receipts
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

func NewFakeL1ChainWithGenesis(genesis *core.Genesis) *FakeL1Chain {
	chain := make([]*types.Block, 0)
	chain = append(chain, genesis.ToBlock())

	firstBlock := types.NewBlockWithHeader(&types.Header{
		ParentHash: genesis.ToBlock().Hash(),
		Root:       genesis.ToBlock().Header().Root,
		Number:     new(big.Int).Add(genesis.ToBlock().Number(), big.NewInt(1)),
		GasLimit:   genesis.ToBlock().GasLimit(),
		Time:       genesis.ToBlock().Time() + 1,
	})
	chain = append(chain, firstBlock)

	return &FakeL1Chain{blocks: chain, blockReceipts: make([]types.Receipts, len(chain))}
}
