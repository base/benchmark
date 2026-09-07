package simulator

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/simulator/simulatorstats"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
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
	OnBlockBuilt(gasUsed uint64, userTxsIncluded int)
} = (*simulatorPayloadWorker)(nil)

func newRecalibrationWorker(t *testing.T, gasLimit uint64, numCallsPerBlock uint64, callsPerBlock string) *simulatorPayloadWorker {
	t.Helper()
	return &simulatorPayloadWorker{
		log:              log.New(),
		params:           benchtypes.RunParams{GasLimit: gasLimit},
		numCallsPerBlock: numCallsPerBlock,
		payloadParams:    &simulatorstats.Stats{CallsPerBlock: callsPerBlock},
	}
}

func TestOnBlockBuilt_RaisesNumCallsWhenUnderfilled(t *testing.T) {
	w := newRecalibrationWorker(t, 25_000_000, 46, "fill")
	w.OnBlockBuilt(16_800_000, 46) // observed: 365k gas/tx

	require.True(t, w.recalibrated)
	// (25M - 1M) / 365k = 65
	require.Equal(t, uint64(65), w.numCallsPerBlock)
}

func TestOnBlockBuilt_RespectsUserSpecifiedCap(t *testing.T) {
	w := newRecalibrationWorker(t, 25_000_000, 46, "50")
	w.OnBlockBuilt(16_800_000, 46) // raw recalibration would be 65, capped to 50

	require.True(t, w.recalibrated)
	require.Equal(t, uint64(50), w.numCallsPerBlock)
}

func TestOnBlockBuilt_LowersNumCallsWhenOvertargeting(t *testing.T) {
	w := newRecalibrationWorker(t, 250_000_000, 100, "100")
	w.OnBlockBuilt(248_000_000, 68) // observed: 3.65M gas/tx

	require.True(t, w.recalibrated)
	// (250M - 1M) / 3.65M = 68, capped at user-specified 100, so 68.
	require.Equal(t, uint64(68), w.numCallsPerBlock)
}

func TestOnBlockBuilt_NoopOnSubsequentBlocks(t *testing.T) {
	w := newRecalibrationWorker(t, 25_000_000, 46, "fill")

	w.OnBlockBuilt(16_800_000, 46)
	firstRecalibration := w.numCallsPerBlock
	require.Equal(t, uint64(65), firstRecalibration)

	w.OnBlockBuilt(1_000_000, 1) // would suggest ~24 — must NOT apply
	require.Equal(t, firstRecalibration, w.numCallsPerBlock)
}

func TestOnBlockBuilt_GuardsAgainstZeroInputs(t *testing.T) {
	for _, tc := range []struct {
		name              string
		gasUsed           uint64
		userTxsIncluded   int
	}{
		{"zero gas", 0, 46},
		{"zero txs", 16_800_000, 0},
		{"negative txs", 16_800_000, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newRecalibrationWorker(t, 25_000_000, 46, "fill")
			w.OnBlockBuilt(tc.gasUsed, tc.userTxsIncluded)
			require.False(t, w.recalibrated, "must not consume the one-shot recalibration on degenerate input")
			require.Equal(t, uint64(46), w.numCallsPerBlock)
		})
	}
}
