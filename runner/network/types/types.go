package types

import (
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/base/base-bench/runner/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

// TestConfig holds all configuration needed for a benchmark test
type TestConfig struct {
	Params     RunParams
	Config     config.Config
	Genesis    core.Genesis
	BatcherKey ecdsa.PrivateKey
	// BatcherAddr is lazily initialized to avoid unnecessary computation
	batcherAddr *common.Address

	PrefundPrivateKey ecdsa.PrivateKey
	PrefundAmount     big.Int
}

// BatcherAddr returns the batcher address, computing it if necessary
func (c *TestConfig) BatcherAddr() common.Address {
	if c.batcherAddr == nil {
		batcherAddr := crypto.PubkeyToAddress(c.BatcherKey.PublicKey)
		c.batcherAddr = &batcherAddr
	}
	return *c.batcherAddr
}

// Params is the parameters for a single benchmark run.
type RunParams struct {
	NodeType       string
	GasLimit       uint64
	PayloadID      string
	BenchmarkRunID string
	Name           string
	Description    string
	BlockTime      time.Duration
	Env            map[string]string
	NumBlocks      int
	Tags           map[string]string
}

func (p RunParams) ToConfig() map[string]interface{} {
	params := map[string]interface{}{
		"NodeType":           p.NodeType,
		"GasLimit":           p.GasLimit,
		"TransactionPayload": p.PayloadID,
		"BenchmarkRun":       p.BenchmarkRunID,
	}

	for k, v := range p.Tags {
		params[k] = v
	}

	return params
}

// ClientOptions applies any client customization options to the given client options.
func (p RunParams) ClientOptions(prevClientOptions config.ClientOptions) config.ClientOptions {
	return prevClientOptions
}
