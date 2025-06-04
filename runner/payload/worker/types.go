package worker

import (
	"context"

	"github.com/base/base-bench/runner/network/mempool"
)

// Note: Payload workers are responsible keeping track of gas in a block and sending transactions to the mempool.
type Worker interface {
	Setup(ctx context.Context) error
	SendTxs(ctx context.Context) error
	Stop(ctx context.Context) error
	Mempool() mempool.FakeMempool
}
