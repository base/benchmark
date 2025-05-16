package proofprogram

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/base/base-bench/runner/logger"
	"github.com/base/base-bench/runner/network/configutil"
	"github.com/base/base-bench/runner/network/proofprogram/fakel1"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type ProofProgram interface {
	Run(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64) error
}

type opProgram struct {
	l2Genesis    *core.Genesis
	log          log.Logger
	opProgramBin string
	l2RPCURL     string
	chain        *fakel1.FakeL1Chain
	batcher      *Batcher
}

func NewOPProgram(genesis *core.Genesis, log log.Logger, opProgramBin string, l2RPCURL string, l1Chain *fakel1.FakeL1Chain, batcherKey *ecdsa.PrivateKey) ProofProgram {
	rollupCfg := configutil.GetRollupConfig(genesis, l1Chain)
	batcher := NewBatcher(rollupCfg, batcherKey, l1Chain)

	return &opProgram{
		l2Genesis:    genesis,
		log:          log,
		opProgramBin: opProgramBin,
		l2RPCURL:     l2RPCURL,
		chain:        l1Chain,
		batcher:      batcher,
	}
}

func (o *opProgram) Run(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64) error {
	// create span batches up to firstTestBlock (exclusive)
	setupPayloads := make([]engine.ExecutableData, firstTestBlock)
	copy(setupPayloads, payloads[:firstTestBlock])
	// create span batches for the rest of the payloads
	payloads = payloads[firstTestBlock:]

	parentHash, err := o.chain.GetLatestBlock()
	if err != nil {
		return fmt.Errorf("failed to get parent hash: %w", err)
	}

	err = o.batcher.CreateAndSendBatch(setupPayloads, parentHash.Hash())
	if err != nil {
		return fmt.Errorf("failed to create span batch: %w", err)
	}

	parentHash, err = o.chain.GetLatestBlock()
	if err != nil {
		return fmt.Errorf("failed to get parent hash: %w", err)
	}

	err = o.batcher.CreateAndSendBatch(payloads, parentHash.Hash())
	if err != nil {
		return fmt.Errorf("failed to create span batch: %w", err)
	}

	l1Proxy := fakel1.NewL1ProxyServer(o.log, 8099, o.chain)

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

	o.log.Info("Fetching L2 head", "number", payloads[0].Number)

	latestL2Block, err := ethClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest l2 block: %w", err)
	}
	o.log.Info("Latest L2 block", "number", latestL2Block.Number, "hash", latestL2Block.Hash().Hex())

	l2Head, err := ethClient.HeaderByNumber(ctx, big.NewInt(int64(payloads[len(payloads)-1].Number)))
	if err != nil {
		return fmt.Errorf("failed to get l2 head: %w", err)
	}
	l2HeadNumber := l2Head.Number

	blockBeforeL2Head, err := ethClient.HeaderByNumber(ctx, new(big.Int).Sub(l2HeadNumber, big.NewInt(1)))
	if err != nil {
		return fmt.Errorf("failed to get block before l2 head: %w", err)
	}

	l2OutputRoot := eth.OutputRoot(&eth.OutputV0{
		StateRoot:                eth.Bytes32(blockBeforeL2Head.Root),
		BlockHash:                blockBeforeL2Head.Hash(),
		MessagePasserStorageRoot: eth.Bytes32(blockBeforeL2Head.WithdrawalsHash.Bytes()),
	})

	// write genesis.json to file locally
	genesisFile, err := os.Create("genesis.json")
	if err != nil {
		return fmt.Errorf("failed to create genesis.json: %w", err)
	}
	defer genesisFile.Close()
	err = json.NewEncoder(genesisFile).Encode(o.l2Genesis)
	if err != nil {
		return fmt.Errorf("failed to encode genesis.json: %w", err)
	}

	// write rollup.json to file locally
	rollupFile, err := os.Create("rollup.json")
	if err != nil {
		return fmt.Errorf("failed to create rollup.json: %w", err)
	}
	defer rollupFile.Close()
	err = json.NewEncoder(rollupFile).Encode(o.batcher.rollupCfg)
	if err != nil {
		return fmt.Errorf("failed to encode rollup.json: %w", err)
	}

	l1Head, err := o.chain.GetBlockByNumber(0)
	if err != nil {
		return fmt.Errorf("failed to get l1 head: %w", err)
	}

	o.chain.PrintChain(o.log)

	// start op-program
	zeroHash := common.Hash{}
	cmd := exec.CommandContext(ctx, o.opProgramBin,
		"--l1", "http://127.0.0.1:8099",
		"--l1.beacon", "http://127.0.0.1:8099",
		"--l1.head", l1Head.Hash().Hex(),
		"--l2", o.l2RPCURL,
		"--l2.head", l2Head.ParentHash.Hex(),
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
