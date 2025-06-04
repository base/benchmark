package types

import (
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

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

var _ ethTypes.BlockType = IsthmusBlockType{}
