package configutil

import (
	"math/big"

	"github.com/base/base-bench/runner/network/proofprogram/fakel1"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

// GetRollupConfig creates a rollup configuration for the given genesis and chain
func GetRollupConfig(genesis *core.Genesis, chain fakel1.L1Chain, batcherAddr common.Address) *rollup.Config {
	var eipParams eth.Bytes8
	copy(eipParams[:], eip1559.EncodeHolocene1559Params(50, 1))

	deltaTime := uint64(0)

	l1Genesis, err := chain.GetBlockByNumber(0)
	if err != nil {
		panic(err)
	}

	rollupCfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1: eth.BlockID{
				Hash:   l1Genesis.Hash(),
				Number: 0,
			},
			L2: eth.BlockID{
				Hash:   genesis.ToBlock().Hash(),
				Number: 0,
			},
			L2Time: genesis.Timestamp,
			SystemConfig: eth.SystemConfig{
				BatcherAddr: batcherAddr,
				Overhead:    eth.Bytes32{0},
				Scalar: eth.EncodeScalar(eth.EcotoneScalars{
					BlobBaseFeeScalar: 0,
					BaseFeeScalar:     0,
				}),
				GasLimit:      params.MaxGasLimit,
				EIP1559Params: eipParams,
				OperatorFeeParams: eth.EncodeOperatorFeeParams(eth.OperatorFeeParams{
					Scalar:   0,
					Constant: 0,
				}),
			},
		},
		BlockTime:               1,
		MaxSequencerDrift:       20,
		SeqWindowSize:           24,
		L1ChainID:               big.NewInt(1),
		DeltaTime:               &deltaTime,
		L2ChainID:               genesis.Config.ChainID,
		RegolithTime:            genesis.Config.RegolithTime,
		CanyonTime:              genesis.Config.CanyonTime,
		EcotoneTime:             genesis.Config.EcotoneTime,
		FjordTime:               genesis.Config.FjordTime,
		GraniteTime:             genesis.Config.GraniteTime,
		HoloceneTime:            genesis.Config.HoloceneTime,
		IsthmusTime:             genesis.Config.IsthmusTime,
		InteropTime:             genesis.Config.InteropTime,
		BatchInboxAddress:       common.Address{1},
		DepositContractAddress:  common.Address{1},
		L1SystemConfigAddress:   common.Address{1},
		ProtocolVersionsAddress: common.Address{1},
		ChannelTimeoutBedrock:   50,
	}
	return rollupCfg
}
