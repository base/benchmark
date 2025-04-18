package payload

import (
	"context"
	"encoding/hex"
	"math/big"
	"os"
	"os/exec"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/clients/proxy"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

type TxFuzzPayloadWorker struct {
	log         log.Logger
	prefundSK   string
	txFuzzBin   string
	elRPCURL    string
	mempool     *mempool.StaticWorkloadMempool
	proxyServer *proxy.ProxyServer
}

func NewTxFuzzPayloadWorker(
	log log.Logger,
	elRPCURL string,
	params benchmark.Params,
	prefundedPrivateKey []byte,
	prefundAmount *big.Int,
	txFuzzBin string,
) (mempool.FakeMempool, Worker, error) {
	mempool := mempool.NewStaticWorkloadMempool(log)
	proxyServer := proxy.NewProxyServer(elRPCURL, log, 8545, mempool)

	t := &TxFuzzPayloadWorker{
		log:         log,
		prefundSK:   hex.EncodeToString(prefundedPrivateKey),
		txFuzzBin:   txFuzzBin,
		elRPCURL:    elRPCURL,
		mempool:     mempool,
		proxyServer: proxyServer,
	}

	return mempool, t, nil
}

// Setup is a no-op for the tx fuzzer
func (t *TxFuzzPayloadWorker) Setup(ctx context.Context) error {
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

func (t *TxFuzzPayloadWorker) Stop(ctx context.Context) error {
	t.proxyServer.Stop()
	return nil
}

func (t *TxFuzzPayloadWorker) SendTxs(ctx context.Context) error {
	t.log.Info("Sending txs in tx-fuzz mode")
	pendingTxs := t.proxyServer.PendingTxs()
	t.proxyServer.ClearPendingTxs()

	t.mempool.AddTransactions(pendingTxs)
	return nil
}
