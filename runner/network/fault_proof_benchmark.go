package network

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
	"github.com/base/base-bench/runner/network/proofprogram"
	"github.com/base/base-bench/runner/network/proofprogram/fakel1"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type ProofProgramBenchmark interface {
	Run(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64) error
}

type opProgramBenchmark struct {
	l2Genesis    *core.Genesis
	log          log.Logger
	opProgramBin string
	l2RPCURL     string
	chain        fakel1.L1Chain
	batcher      *proofprogram.Batcher
	rollupCfg    *rollup.Config
}

func NewOPProgramBenchmark(genesis *core.Genesis, log log.Logger, opProgramBin string, l2RPCURL string, l1Chain fakel1.L1Chain, batcherKey *ecdsa.PrivateKey) ProofProgramBenchmark {
	rollupCfg := configutil.GetRollupConfig(genesis, l1Chain, crypto.PubkeyToAddress(batcherKey.PublicKey))
	batcher := proofprogram.NewBatcher(rollupCfg, batcherKey, l1Chain)

	return &opProgramBenchmark{
		l2Genesis:    genesis,
		log:          log,
		opProgramBin: opProgramBin,
		l2RPCURL:     l2RPCURL,
		chain:        l1Chain,
		rollupCfg:    rollupCfg,
		batcher:      batcher,
	}
}

func (o *opProgramBenchmark) Run(ctx context.Context, payloads []engine.ExecutableData, firstTestBlock uint64) error {
	// Split payloads into setup and test groups
	setupPayloads := make([]engine.ExecutableData, firstTestBlock)
	copy(setupPayloads, payloads[:firstTestBlock])
	testPayloads := payloads[firstTestBlock:]

	// Process batches
	if err := o.processBatches(setupPayloads, testPayloads); err != nil {
		return err
	}

	// Start L1 proxy server
	l1Proxy := fakel1.NewL1ProxyServer(o.log, 8099, o.chain)
	if err := l1Proxy.Run(ctx); err != nil {
		return fmt.Errorf("failed to start l1 proxy: %w", err)
	}
	defer l1Proxy.Stop()

	// Connect to L2 RPC and get block information
	ethClient, err := o.connectToL2RPC(ctx)
	if err != nil {
		return err
	}

	// Prepare for op-program execution
	l2HeadNumber := testPayloads[len(testPayloads)-1].Number
	blockBeforeL2Head, l2OutputRoot, claimOutputRoot, err := o.prepareBlockData(ctx, ethClient, l2HeadNumber)
	if err != nil {
		return err
	}

	// Write necessary files
	if err := o.writeConfigFiles(); err != nil {
		return err
	}

	// Execute op-program
	return o.executeOpProgram(ctx, blockBeforeL2Head, l2HeadNumber, l2OutputRoot, claimOutputRoot)
}

// processBatches creates and sends both setup and test batches
func (o *opProgramBenchmark) processBatches(setupPayloads, testPayloads []engine.ExecutableData) error {
	// Process setup batches
	parentHash, err := o.chain.GetLatestBlock()
	if err != nil {
		return fmt.Errorf("failed to get parent hash: %w", err)
	}

	if err := o.batcher.CreateAndSendBatch(setupPayloads, parentHash.Hash()); err != nil {
		return fmt.Errorf("failed to create span batch for setup: %w", err)
	}

	// Process test batches
	parentHash, err = o.chain.GetLatestBlock()
	if err != nil {
		return fmt.Errorf("failed to get parent hash: %w", err)
	}

	if err := o.batcher.CreateAndSendBatch(testPayloads, parentHash.Hash()); err != nil {
		return fmt.Errorf("failed to create span batch for test: %w", err)
	}

	return nil
}

// connectToL2RPC establishes a connection to the L2 RPC endpoint
func (o *opProgramBenchmark) connectToL2RPC(ctx context.Context) (*ethclient.Client, error) {
	o.log.Info("Dialing L2 RPC", "url", o.l2RPCURL)

	rpcClient, err := rpc.DialOptions(ctx, o.l2RPCURL, rpc.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	}))
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial rpc")
	}

	return ethclient.NewClient(rpcClient), nil
}

// prepareBlockData fetches necessary block data for op-program execution
func (o *opProgramBenchmark) prepareBlockData(ctx context.Context, ethClient *ethclient.Client, l2HeadNumber uint64) (*types.Header, eth.Bytes32, eth.Bytes32, error) {
	// Fetch latest L2 block
	latestL2Block, err := ethClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, eth.Bytes32{}, eth.Bytes32{}, fmt.Errorf("failed to get latest l2 block: %w", err)
	}
	o.log.Info("Latest L2 block", "number", latestL2Block.Number, "hash", latestL2Block.Hash().Hex())

	// Get block before L2 head
	blockBeforeL2Head, err := ethClient.HeaderByNumber(ctx, big.NewInt(int64(l2HeadNumber-1)))
	if err != nil {
		return nil, eth.Bytes32{}, eth.Bytes32{}, fmt.Errorf("failed to get block before l2 head %d: %w", big.NewInt(int64(l2HeadNumber-1)), err)
	}

	// Calculate L2 output root
	l2OutputRoot := eth.OutputRoot(&eth.OutputV0{
		StateRoot:                eth.Bytes32(blockBeforeL2Head.Root),
		BlockHash:                blockBeforeL2Head.Hash(),
		MessagePasserStorageRoot: eth.Bytes32(blockBeforeL2Head.WithdrawalsHash.Bytes()),
	})

	// Get expected claim block
	expectedClaimBlock, err := ethClient.BlockByNumber(ctx, big.NewInt(int64(l2HeadNumber)))
	if err != nil {
		return nil, eth.Bytes32{}, eth.Bytes32{}, fmt.Errorf("failed to get expected claim block %d: %w", l2HeadNumber, err)
	}

	if expectedClaimBlock == nil {
		return nil, eth.Bytes32{}, eth.Bytes32{}, fmt.Errorf("expected claim block %d not found", l2HeadNumber)
	}

	// Calculate claim output root
	claimOutputRoot := eth.OutputRoot(&eth.OutputV0{
		StateRoot:                eth.Bytes32(expectedClaimBlock.Root()),
		BlockHash:                expectedClaimBlock.Hash(),
		MessagePasserStorageRoot: eth.Bytes32(*expectedClaimBlock.WithdrawalsRoot()),
	})

	return blockBeforeL2Head, l2OutputRoot, claimOutputRoot, nil
}

// writeConfigFiles writes necessary configuration files to disk
func (o *opProgramBenchmark) writeConfigFiles() error {
	// Write rollup.json
	rollupFile, err := os.Create("rollup.json")
	if err != nil {
		return fmt.Errorf("failed to create rollup.json: %w", err)
	}
	defer func() { _ = rollupFile.Close() }()

	if err = json.NewEncoder(rollupFile).Encode(o.rollupCfg); err != nil {
		return fmt.Errorf("failed to encode rollup.json: %w", err)
	}

	return nil
}

// executeOpProgram runs the op-program binary with the necessary arguments
func (o *opProgramBenchmark) executeOpProgram(ctx context.Context, blockBeforeL2Head *types.Header, l2HeadNumber uint64, l2OutputRoot, claimOutputRoot eth.Bytes32) error {
	l1Head, err := o.chain.GetBlockByNumber(3)
	if err != nil {
		return fmt.Errorf("failed to get l1 head: %w", err)
	}

	// Start op-program
	cmd := exec.CommandContext(ctx, o.opProgramBin,
		"--l1", "http://127.0.0.1:8099",
		"--l1.beacon", "http://127.0.0.1:8099",
		"--l2", o.l2RPCURL,
		"--l1.head", l1Head.Hash().Hex(),
		"--l2.head", blockBeforeL2Head.Hash().Hex(),
		"--l2.outputroot", common.Hash(l2OutputRoot).Hex(),
		"--l2.blocknumber", fmt.Sprintf("%d", l2HeadNumber),
		"--l2.claim", common.Hash(claimOutputRoot).Hex(),
		"--l2.genesis", "genesis.json",
		"--rollup.config", "rollup.json",
	)

	cmd.Stdout = logger.NewLogWriterWithLevel(o.log, slog.LevelInfo)
	cmd.Stderr = logger.NewLogWriterWithLevel(o.log, slog.LevelInfo)

	if err = cmd.Run(); err != nil {
		o.logFailureDetails(l2HeadNumber, blockBeforeL2Head, claimOutputRoot, l2OutputRoot)
		return fmt.Errorf("failed to run op-program: %w", err)
	}

	return nil
}

// logFailureDetails logs detailed information when op-program execution fails
func (o *opProgramBenchmark) logFailureDetails(l2HeadNumber uint64, blockBeforeL2Head *types.Header, claimOutputRoot, l2OutputRoot eth.Bytes32) {
	o.log.Info("op-program execution failed",
		"l2HeadNumber", l2HeadNumber,
		"blockBeforeL2Head", blockBeforeL2Head.Hash().Hex(),
		"claimOutputRoot", common.Hash(claimOutputRoot).Hex(),
		"l2OutputRoot", common.Hash(l2OutputRoot).Hex(),
	)
}
