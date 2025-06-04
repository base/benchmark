package worker

import (
	"context"
	"log"
	"math/big"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/network/mempool"
)

// Note: Payload workers are responsible keeping track of gas in a block and sending transactions to the mempool.
type Worker interface {
	Setup(ctx context.Context) error
	SendTxs(ctx context.Context) error
	Stop(ctx context.Context) error
	Mempool() mempool.FakeMempool
}

type NewWorkerFn func(logger log.Logger, elRPCURL string, params benchmark.Params, prefundedPrivateKey []byte, prefundAmount *big.Int) (Worker, error)
