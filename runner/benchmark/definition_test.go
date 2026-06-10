package benchmark_test

import (
	"testing"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/stretchr/testify/require"
)

func TestFlashblocksLeewayTime(t *testing.T) {
	t.Run("configured", func(t *testing.T) {
		config := &benchmark.BenchmarkConfig{
			Flashblocks: &benchmark.FlashblocksConfig{
				LeewayTime: "300",
			},
		}

		require.Equal(t, "300", config.FlashblocksLeewayTime())
	})

	t.Run("default", func(t *testing.T) {
		config := &benchmark.BenchmarkConfig{}

		require.Empty(t, config.FlashblocksLeewayTime())
	})
}
