package blocks

import "github.com/ethereum/go-ethereum/core/types"

// BasicBlockType implements what chain config would usually implement.
type IsthmusBlockType struct{}

// HasOptimismWithdrawalsRoot implements types.BlockType.
func (b IsthmusBlockType) HasOptimismWithdrawalsRoot(blkTime uint64) bool {
	return true
}

// IsIsthmus implements types.BlockType.
func (b IsthmusBlockType) IsIsthmus(blkTime uint64) bool {
	return true
}

var _ types.BlockType = IsthmusBlockType{}

// BasicBlockType implements what chain config would usually implement.
type L1BlockType struct{}

// HasOptimismWithdrawalsRoot implements types.BlockType.
func (b L1BlockType) HasOptimismWithdrawalsRoot(blkTime uint64) bool {
	return false
}

// IsIsthmus implements types.BlockType.
func (b L1BlockType) IsIsthmus(blkTime uint64) bool {
	return false
}

var _ types.BlockType = L1BlockType{}
