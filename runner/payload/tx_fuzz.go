package payload

import (
	"context"
	"encoding/hex"
	"math/big"
	"os/exec"
	"time"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/logger"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// TxFuzzPayloadWorker executes a transaction fuzzer binary
type TxFuzzPayloadWorker struct {
	log       log.Logger
	prefundSK string
	txFuzzBin string
	elRPCURL  string
	mempool   *mempool.StaticWorkloadMempool
}

// NewTxFuzzPayloadWorker creates a new tx fuzzer payload worker
func NewTxFuzzPayloadWorker(
	log log.Logger,
	elRPCURL string,
	params benchmark.Params,
	prefundedPrivateKey []byte,
	prefundAmount *big.Int,
	txFuzzBin string,
) (mempool.FakeMempool, Worker, error) {
	mempool := mempool.NewStaticWorkloadMempool(params.GasLimit)

	return mempool, &TxFuzzPayloadWorker{
		log:       log,
		prefundSK: hex.EncodeToString(prefundedPrivateKey),
		txFuzzBin: txFuzzBin,
		elRPCURL:  elRPCURL,
		mempool:   mempool,
	}, nil
}

// Setup is a no-op for the tx fuzzer
func (t *TxFuzzPayloadWorker) Setup(ctx context.Context) error {
	// No setup needed
	// proxy := proxy.NewProxyServer(t.mempool, t.log, t.config.ProxyPort())
	return nil
}

// Run executes the transaction fuzzer
func (t *TxFuzzPayloadWorker) Run(ctx context.Context) error {
	t.log.Info("Starting tx fuzzer", "binary", t.txFuzzBin)

	// Run for a limited number of blocks just like TransferOnlyPayloadWorker
	numBlocks := 10
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Create the command but don't start it yet
	cmd := exec.CommandContext(ctx, t.txFuzzBin, "spam", "--sk", t.prefundSK, "--rpc", t.elRPCURL, "--slot-time", "1")
	stdoutLogger := logger.NewLogWriter(t.log)
	stderrLogger := logger.NewLogWriter(t.log)

	cmd.Stdout = stdoutLogger
	cmd.Stderr = stderrLogger

	// Start the tx fuzzer
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start tx fuzzer")
	}

	t.log.Info("Started tx fuzzer process", "pid", cmd.Process.Pid)

	// Run for specified number of blocks
	for i := 0; i < numBlocks; i++ {
		select {
		case <-ctx.Done():
			// Context canceled, kill the process and return
			if cmd.Process != nil {
				err := cmd.Process.Kill()
				if err != nil {
					t.log.Warn("Failed to kill tx fuzzer process", "err", err)
				}
			}
			return ctx.Err()
		case <-ticker.C:
			// Tick - wait for next block
		}
	}

	// After numBlocks, terminate the process
	if cmd.Process != nil {
		t.log.Info("Terminating tx fuzzer after completing blocks", "numBlocks", numBlocks)
		if err := cmd.Process.Kill(); err != nil {
			t.log.Warn("Failed to kill tx fuzzer process", "err", err)
		}
	}

	return nil
}
