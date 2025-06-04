package network

import (
	"fmt"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/base/base-bench/runner/network/proofprogram/fakel1"
	"github.com/base/base-bench/runner/network/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type l1Chain struct {
	chain fakel1.L1Chain
}

func newL1Chain(config *types.TestConfig) (*l1Chain, error) {
	chain, err := setupChain(config)
	if err != nil {
		return nil, fmt.Errorf("failed to setup L1 chain: %w", err)
	}

	return &l1Chain{
		chain: chain,
	}, nil
}

func makeL1Genesis(prefundAddr []common.Address) core.Genesis {
	zero := uint64(0)
	alloc := make(ethTypes.GenesisAlloc)
	for _, addr := range prefundAddr {
		alloc[addr] = ethTypes.Account{
			Balance: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(params.Ether)),
		}
	}
	blobSchedule := *params.DefaultBlobSchedule
	l1Genesis := core.Genesis{
		Config: &params.ChainConfig{
			ChainID:             big.NewInt(1),
			HomesteadBlock:      big.NewInt(0),
			DAOForkBlock:        nil,
			DAOForkSupport:      false,
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			MuirGlacierBlock:    big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			ArrowGlacierBlock:   big.NewInt(0),
			GrayGlacierBlock:    big.NewInt(0),
			ShanghaiTime:        &zero,
			CancunTime:          &zero,
			PragueTime:          &zero,
			// To enable post-Merge consensus at genesis
			MergeNetsplitBlock:      big.NewInt(0),
			TerminalTotalDifficulty: big.NewInt(0),
			// use default Ethereum prod blob schedules
			BlobScheduleConfig: &blobSchedule,
		},
		Nonce:      0,
		Alloc:      alloc,
		Timestamp:  0, // blocks will have better timestamps
		ExtraData:  []byte{},
		GasLimit:   30_000_000,
		Difficulty: big.NewInt(0),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		BaseFee:    big.NewInt(1e9),
	}

	return l1Genesis
}

func setupChain(config *types.TestConfig) (fakel1.L1Chain, error) {
	blobsFolder := path.Join(config.Config.DataDir(), "blobs")
	if err := os.MkdirAll(blobsFolder, 0755); err != nil {
		return nil, fmt.Errorf("failed to create blobs folder: %w", err)
	}

	prefundAccts := []common.Address{
		config.BatcherAddr(),
	}

	// use current time as the timestamp to base the L1 chain on
	l1Genesis := makeL1Genesis(prefundAccts)
	l2FirstBlockTime := uint64(time.Now().Add(-time.Minute).Unix())

	chain, err := fakel1.NewFakeL1ChainWithGenesis(blobsFolder, &l1Genesis, l2FirstBlockTime)
	if err != nil {
		return nil, fmt.Errorf("failed to make chain: %w", err)
	}

	return chain, nil
}
