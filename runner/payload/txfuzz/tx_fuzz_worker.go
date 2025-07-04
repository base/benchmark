package txfuzz

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"os"
	"os/exec"

	"github.com/base/base-bench/runner/clients/common/proxy"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

type txFuzzPayloadWorker struct {
	log         log.Logger
	prefundSK   string
	txFuzzBin   string
	elRPCURL    string
	mempool     *mempool.StaticWorkloadMempool
	proxyServer *proxy.ProxyServer
}

type TxFuzzPayloadDefinition struct {
}

func NewTxFuzzPayloadWorker(
	log log.Logger,
	elRPCURL string,
	params types.RunParams,
	prefundedPrivateKey ecdsa.PrivateKey,
	prefundAmount *big.Int,
	txFuzzBin string,
	chainID *big.Int,
) (worker.Worker, error) {
	mempool := mempool.NewStaticWorkloadMempool(log, chainID)
	proxyServer := proxy.NewProxyServer(elRPCURL, log, 8545, mempool)

	t := &txFuzzPayloadWorker{
		log:         log,
		prefundSK:   hex.EncodeToString(prefundedPrivateKey.D.Bytes()),
		txFuzzBin:   txFuzzBin,
		elRPCURL:    elRPCURL,
		mempool:     mempool,
		proxyServer: proxyServer,
	}

	return t, nil
}

func (t *txFuzzPayloadWorker) Mempool() mempool.FakeMempool {
	return t.mempool
}

// Setup is a no-op for the tx fuzzer
func (t *txFuzzPayloadWorker) Setup(ctx context.Context) error {
	err := t.proxyServer.Run(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to run proxy server")
	}

	t.log.Info("Sending txs in tx-fuzz mode")

	cmd := exec.CommandContext(ctx, t.txFuzzBin, "spam", "--sk", t.prefundSK, "--rpc", t.elRPCURL, "--slot-time", "1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start tx fuzzer")
	}

	return nil
}

func (t *txFuzzPayloadWorker) Stop(ctx context.Context) error {
	t.proxyServer.Stop()
	return nil
}

func (t *txFuzzPayloadWorker) SendTxs(ctx context.Context) error {
	t.log.Info("Sending txs in tx-fuzz mode")
	pendingTxs := t.proxyServer.PendingTxs()
	t.proxyServer.ClearPendingTxs()

	t.mempool.AddTransactions(pendingTxs)
	return nil
}
