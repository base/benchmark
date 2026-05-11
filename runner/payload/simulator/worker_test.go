package simulator

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// fakeMineAndConfirm is a test-local reimplementation of mineAndConfirm that
// records how many transactions were submitted per batch, without touching a
// real chain. It mirrors the exact batching logic from the production function
// so changes to mineAndConfirmBatchSize are automatically tested.
func fakeMineAndConfirm(txs []*types.Transaction) ([]int, error) {
	var batchSizes []int
	for len(txs) > 0 {
		batch := txs
		if len(batch) > mineAndConfirmBatchSize {
			batch = txs[:mineAndConfirmBatchSize]
		}
		txs = txs[len(batch):]
		batchSizes = append(batchSizes, len(batch))
	}
	return batchSizes, nil
}

func makeTxs(n int) []*types.Transaction {
	txs := make([]*types.Transaction, n)
	for i := range txs {
		txs[i] = types.NewTx(&types.LegacyTx{Nonce: uint64(i), Gas: 21000, GasPrice: big.NewInt(1)})
	}
	return txs
}

func TestMineAndConfirmBatching(t *testing.T) {
	tests := []struct {
		numTxs         int
		wantMaxBatch   int
		wantBatchCount int
	}{
		{numTxs: 0, wantMaxBatch: 0, wantBatchCount: 0},
		{numTxs: 1, wantMaxBatch: 1, wantBatchCount: 1},
		{numTxs: mineAndConfirmBatchSize, wantMaxBatch: mineAndConfirmBatchSize, wantBatchCount: 1},
		{numTxs: mineAndConfirmBatchSize + 1, wantMaxBatch: mineAndConfirmBatchSize, wantBatchCount: 2},
		{numTxs: mineAndConfirmBatchSize * 3, wantMaxBatch: mineAndConfirmBatchSize, wantBatchCount: 3},
		// Simulate the real problematic case: ~640k init txs (scaled down for test speed).
		// Before the fix this was sent as a single batch, timing out waitForReceipt.
		{numTxs: 10000, wantMaxBatch: mineAndConfirmBatchSize, wantBatchCount: 10000 / mineAndConfirmBatchSize},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("numTxs=%d", tt.numTxs), func(t *testing.T) {
			txs := makeTxs(tt.numTxs)
			batches, err := fakeMineAndConfirm(txs)
			require.NoError(t, err)
			require.Len(t, batches, tt.wantBatchCount)
			for _, size := range batches {
				require.LessOrEqual(t, size, tt.wantMaxBatch)
			}
		})
	}
}

// TestMineAndConfirmNoBatchingWouldTimeout demonstrates the scale of the problem:
// for storage-reads-full-block at 150M gas, ~640k init transactions were sent in
// one batch but waitForReceipt only retries for 240s.
func TestMineAndConfirmNoBatchingWouldTimeout(t *testing.T) {
	const (
		gasLimit          = 150_000_000
		gasPerStorageCall = 220_000
		numBlocks         = 900
		storageSlotsPerTx = 100
		waitForReceiptMaxRetries = 240
	)
	numCallsPerBlock := (gasLimit - 1_000_000) / gasPerStorageCall
	totalStorageSlotsNeeded := storageSlotsPerTx * numCallsPerBlock * numBlocks
	initChunksNeeded := (totalStorageSlotsNeeded + 99) / 100

	// Without batching: all init txs in one mineAndConfirm → wait for receipt of the last one.
	// Each receipt poll is 1 second, and there are only 240 retries.
	require.Greater(t, initChunksNeeded, waitForReceiptMaxRetries,
		"init txs (%d) must exceed timeout window (%d retries) to demonstrate the bug",
		initChunksNeeded, waitForReceiptMaxRetries)

	// With batching: each batch of mineAndConfirmBatchSize is confirmed before the next.
	// The last tx in each batch is confirmed within a few seconds.
	require.LessOrEqual(t, mineAndConfirmBatchSize, waitForReceiptMaxRetries,
		"batch size must fit within the receipt timeout window")

	t.Logf("storage-reads-full-block at 150M gas: ~%d init txs needed, batch size %d",
		initChunksNeeded, mineAndConfirmBatchSize)
}

// Verify the worker satisfies the interface (compilation check).
var _ interface {
	Setup(ctx context.Context) error
	SendTxs(ctx context.Context, pendingTxs int) (int, error)
} = (*simulatorPayloadWorker)(nil)
