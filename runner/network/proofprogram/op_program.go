package proofprogram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/base/base-bench/runner/logger"
	"github.com/base/base-bench/runner/network/consensus"
	"github.com/base/base-bench/runner/network/proofprogram/fakel1"
	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derive_params "github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type ProofProgram interface {
	Run(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64) error
}

type opProgram struct {
	genesis      *core.Genesis
	log          log.Logger
	opProgramBin string
	l2RPCURL     string

	chain *fakel1.FakeL1Chain
}

func makeChain() *fakel1.FakeL1Chain {
	l1Genesis := core.Genesis{
		Config: &params.ChainConfig{
			ChainID: big.NewInt(1),
		},
	}

	return fakel1.NewFakeL1ChainWithGenesis(&l1Genesis)
}

func NewOPProgram(genesis *core.Genesis, log log.Logger, opProgramBin string, l2RPCURL string) ProofProgram {
	return &opProgram{
		genesis:      genesis,
		log:          log,
		opProgramBin: opProgramBin,
		l2RPCURL:     l2RPCURL,
		chain:        makeChain(),
	}
}

func (o *opProgram) getRollupConfig() *rollup.Config {
	var eipParams eth.Bytes8
	copy(eipParams[:], eip1559.EncodeHolocene1559Params(50, 1))

	deltaTime := uint64(0)

	l1Genesis, err := o.chain.GetBlockByNumber(0)
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
				Hash:   o.genesis.ToBlock().Hash(), // TODO: snapshot support
				Number: 0,
			},
			L2Time: o.genesis.Timestamp,
			SystemConfig: eth.SystemConfig{
				BatcherAddr: common.Address{1},
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
		BlockTime:               1, // TODO?
		MaxSequencerDrift:       20,
		SeqWindowSize:           24,
		L1ChainID:               big.NewInt(1),
		DeltaTime:               &deltaTime,
		L2ChainID:               o.genesis.Config.ChainID,
		RegolithTime:            o.genesis.Config.RegolithTime,
		CanyonTime:              o.genesis.Config.CanyonTime,
		EcotoneTime:             o.genesis.Config.EcotoneTime,
		FjordTime:               o.genesis.Config.FjordTime,
		GraniteTime:             o.genesis.Config.GraniteTime,
		HoloceneTime:            o.genesis.Config.HoloceneTime,
		IsthmusTime:             o.genesis.Config.IsthmusTime,
		InteropTime:             o.genesis.Config.InteropTime,
		BatchInboxAddress:       common.Address{1},
		DepositContractAddress:  common.Address{1},
		L1SystemConfigAddress:   common.Address{1},
		ProtocolVersionsAddress: common.Address{1},
		ChannelTimeoutBedrock:   50,
	}
	return rollupCfg
}

func (o *opProgram) createSpanBatch(payloads []engine.ExecutableData) ([]byte, error) {
	// generate batches based on the payloads
	target := batcher.MaxDataSize(1, 20000000)

	cfg := o.getRollupConfig()

	chainSpec := rollup.NewChainSpec(cfg)
	ch, err := derive.NewSpanChannelOut(target, derive.Zlib, chainSpec)
	// use singular batches in all other cases

	if err != nil {
		return nil, fmt.Errorf("failed to create span channel: %w", err)
	}

	for _, payload := range payloads {
		block, err := engine.ExecutableDataToBlock(payload, []common.Hash{}, &common.Hash{}, [][]byte{}, consensus.IsthmusBlockType{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert payload to block")
		}

		_, err = ch.AddBlock(cfg, block)
		if err != nil {
			return nil, fmt.Errorf("failed to add block to channel: %w", err)
		}

	}

	err = ch.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close channel: %w", err)
	}

	data := new(bytes.Buffer)
	data.WriteByte(derive_params.DerivationVersion0)

	if _, err := ch.OutputFrame(data, 20000000-1); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to output frame: %w", err)
	}

	log.Info("Generated batch", "size", data.Len())

	return data.Bytes(), nil
}

func (o *opProgram) Run(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64) error {
	_, err := o.createSpanBatch(payloads)
	if err != nil {
		return fmt.Errorf("failed to create span batch: %w", err)
	}

	l1Proxy := fakel1.NewL1ProxyServer(o.log, 8099, o.chain, o.genesis.Config)

	err = l1Proxy.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to start l1 proxy: %w", err)
	}
	defer l1Proxy.Stop()

	o.log.Info("Dialing L2 RPC", "url", o.l2RPCURL)

	rpcClient, err := rpc.DialOptions(ctx, o.l2RPCURL, rpc.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	}))
	if err != nil {
		return errors.Wrap(err, "failed to dial rpc")
	}

	ethClient := ethclient.NewClient(rpcClient)

	o.log.Info("Fetching L2 head", "number", payloads[firstTestBlock].Number)

	latestL2Block, err := ethClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest l2 block: %w", err)
	}
	o.log.Info("Latest L2 block", "number", latestL2Block.Number, "hash", latestL2Block.Hash().Hex())

	l2Head, err := ethClient.HeaderByNumber(ctx, big.NewInt(int64(payloads[firstTestBlock].Number)))
	if err != nil {
		return fmt.Errorf("failed to get l2 head: %w", err)
	}
	l2HeadHash := l2Head.Hash()
	l2HeadNumber := l2Head.Number

	l2OutputRoot := eth.OutputRoot(&eth.OutputV0{
		StateRoot:                eth.Bytes32(l2Head.Root),
		BlockHash:                l2Head.Hash(),
		MessagePasserStorageRoot: eth.Bytes32(l2Head.WithdrawalsHash.Bytes()),
	})

	// write genesis.json to file locally
	genesisFile, err := os.Create("genesis.json")
	if err != nil {
		return fmt.Errorf("failed to create genesis.json: %w", err)
	}
	defer genesisFile.Close()
	err = json.NewEncoder(genesisFile).Encode(o.genesis)
	if err != nil {
		return fmt.Errorf("failed to encode genesis.json: %w", err)
	}

	// write rollup.json to file locally
	rollupCfg := o.getRollupConfig()
	rollupFile, err := os.Create("rollup.json")
	if err != nil {
		return fmt.Errorf("failed to create rollup.json: %w", err)
	}
	defer rollupFile.Close()
	err = json.NewEncoder(rollupFile).Encode(rollupCfg)
	if err != nil {
		return fmt.Errorf("failed to encode rollup.json: %w", err)
	}

	l1Head, err := o.chain.GetBlockByNumber(1)
	if err != nil {
		return fmt.Errorf("failed to get l1 head: %w", err)
	}

	// start op-program
	zeroHash := common.Hash{}
	cmd := exec.CommandContext(ctx, o.opProgramBin,
		"--l1", "http://127.0.0.1:8099",
		"--l1.beacon", "http://127.0.0.1:8099",
		"--l1.head", l1Head.Hash().Hex(),
		"--l2", o.l2RPCURL,
		"--l2.head", l2HeadHash.Hex(),
		"--l2.blocknumber", l2HeadNumber.String(),
		"--l2.claim", zeroHash.Hex(),
		"--l2.outputroot", common.Hash(l2OutputRoot).Hex(),
		"--l2.genesis", "genesis.json",
		"--rollup.config", "rollup.json",
	)

	cmd.Stdout = logger.NewLogWriterWithLevel(o.log, slog.LevelInfo)
	cmd.Stderr = logger.NewLogWriterWithLevel(o.log, slog.LevelInfo)

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("failed to run op-program: %w", err)
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("op-program exited with error: %w", err)
	}

	return nil
}
