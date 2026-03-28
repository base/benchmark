package worker

import (
	"context"

	"github.com/base/base-bench/runner/network/mempool"
)

// Note: Payload workers are responsible keeping track of gas in a block and sending transactions to the mempool.
type Worker interface {
	Setup(ctx context.Context) error
	// SendTxs generates and queues transactions for the next block.
	// pendingTxs is the number of previously-sent transactions still in the node's
	// mempool; implementations should reduce their output accordingly so the
	// mempool stays close to one block's worth of work.
	// Returns the number of transactions actually queued.
	SendTxs(ctx context.Context, pendingTxs int) (int, error)
	Stop(ctx context.Context) error
	Mempool() mempool.FakeMempool
}
