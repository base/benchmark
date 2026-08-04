package simulatorstats

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpcodeStatsAdd_UnionOfKeys(t *testing.T) {
	a := OpcodeStats{"A": 1, "B": 2}
	b := OpcodeStats{"B": 10, "C": 100}

	got := a.Add(b)

	require.Equal(t, 1.0, got["A"], "key only in receiver must be preserved")
	require.Equal(t, 12.0, got["B"], "shared key must sum")
	require.Equal(t, 100.0, got["C"], "key only in arg must be preserved")
	require.Len(t, got, 3)
}

func TestOpcodeStatsAdd_EmptyOther(t *testing.T) {
	a := OpcodeStats{"A": 1, "B": 2}
	got := a.Add(OpcodeStats{})
	require.Equal(t, 1.0, got["A"])
	require.Equal(t, 2.0, got["B"])
	require.Len(t, got, 2)
}

func TestOpcodeStatsAdd_EmptyReceiver(t *testing.T) {
	got := OpcodeStats{}.Add(OpcodeStats{"A": 1, "B": 2})
	require.Equal(t, 1.0, got["A"])
	require.Equal(t, 2.0, got["B"])
	require.Len(t, got, 2)
}

func TestOpcodeStatsSub_UnionOfKeys(t *testing.T) {
	a := OpcodeStats{"A": 10, "B": 20}
	b := OpcodeStats{"B": 5, "C": 100}

	got := a.Sub(b)

	require.Equal(t, 10.0, got["A"], "key only in receiver must be preserved")
	require.Equal(t, 15.0, got["B"], "shared key must subtract")
	require.Equal(t, -100.0, got["C"], "key only in arg must be included (negated)")
	require.Len(t, got, 3)
}

func TestOpcodeStatsSub_EmptyOther(t *testing.T) {
	a := OpcodeStats{"A": 10, "B": 20}
	got := a.Sub(OpcodeStats{})
	require.Equal(t, 10.0, got["A"])
	require.Equal(t, 20.0, got["B"])
	require.Len(t, got, 2)
}

func TestStatsSubAdd_FirstTxBlockCountsIncludePrecompiles(t *testing.T) {
	base := &Stats{
		Precompiles: OpcodeStats{"ecrecover": 0.5, "bls12381MapG2": 1.0},
		Opcodes:     OpcodeStats{"KECCAK256": 10.0},
	}

	expected := base.Mul(1.0)
	actual := NewStats()

	blockCounts := expected.Sub(actual).Round()

	require.Equal(t, 1.0, blockCounts.Precompiles["ecrecover"],
		"precompiles missing in blockCounts means worker txs skip precompile execution")
	require.Equal(t, 1.0, blockCounts.Precompiles["bls12381MapG2"])
	require.Equal(t, 10.0, blockCounts.Opcodes["KECCAK256"])

	actual = actual.Add(blockCounts)
	require.Equal(t, 1.0, actual.Precompiles["ecrecover"],
		"accumulated actual must remember the keys we added")
	require.Equal(t, 1.0, actual.Precompiles["bls12381MapG2"])
}
