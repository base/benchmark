package benchmark_test

import (
	"testing"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/stretchr/testify/require"
)

func TestResolveTestRunsFromMatrix(t *testing.T) {
	tests := []struct {
		name    string
		config  benchmark.TestDefinition
		want    []benchmark.TestRun
		wantErr bool
	}{
		{
			name: "basic config with single value",
			config: benchmark.TestDefinition{
				Variables: []benchmark.Param{
					{
						ParamType: benchmark.ParamTypeTxWorkload,
						Value:     stringPtr("simple"),
					},
				},
			},
			want: []benchmark.TestRun{
				{
					Params: benchmark.Params{
						NodeType:           "geth",
						TransactionPayload: benchmark.TransactionPayload("simple"),
						GasLimit:           benchmark.DefaultParams.GasLimit,
						BlockTime:          benchmark.DefaultParams.BlockTime,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "config with multiple values",
			config: benchmark.TestDefinition{
				Variables: []benchmark.Param{
					{
						ParamType: benchmark.ParamTypeTxWorkload,
						Values:    []interface{}{"simple", "complex"},
					},
					{
						ParamType: benchmark.ParamTypeNode,
						Values:    []interface{}{"geth", "erigon"},
					},
				},
			},
			want: []benchmark.TestRun{
				{
					Params: benchmark.Params{
						NodeType:           "geth",
						GasLimit:           benchmark.DefaultParams.GasLimit,
						TransactionPayload: benchmark.TransactionPayload("simple"),
						BlockTime:          benchmark.DefaultParams.BlockTime,
					},
				},
				{
					Params: benchmark.Params{
						NodeType:           "erigon",
						GasLimit:           benchmark.DefaultParams.GasLimit,
						TransactionPayload: benchmark.TransactionPayload("simple"),
						BlockTime:          benchmark.DefaultParams.BlockTime,
					},
				},
				{
					Params: benchmark.Params{
						NodeType:           "geth",
						GasLimit:           benchmark.DefaultParams.GasLimit,
						TransactionPayload: benchmark.TransactionPayload("complex"),
						BlockTime:          benchmark.DefaultParams.BlockTime,
					},
				},
				{
					Params: benchmark.Params{
						NodeType:           "erigon",
						GasLimit:           benchmark.DefaultParams.GasLimit,
						TransactionPayload: benchmark.TransactionPayload("complex"),
						BlockTime:          benchmark.DefaultParams.BlockTime,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate param type",
			config: benchmark.TestDefinition{
				Variables: []benchmark.Param{
					{
						ParamType: benchmark.ParamTypeTxWorkload,
						Value:     stringPtr("simple"),
					},
					{
						ParamType: benchmark.ParamTypeTxWorkload,
						Value:     stringPtr("complex"),
					},
				},
			},
			want:    []benchmark.TestRun{},
			wantErr: true,
		},
		{
			name: "missing transaction payload",
			config: benchmark.TestDefinition{
				Variables: []benchmark.Param{
					{
						ParamType: benchmark.ParamTypeNode,
						Value:     stringPtr("geth"),
					},
				},
			},
			want:    []benchmark.TestRun{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := benchmark.ResolveTestRunsFromMatrix(tt.config, "")

			if tt.wantErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}
			// ignore outputDir
			for i := range tt.want {
				tt.want[i].OutputDir = ""
			}
			for i := range got {
				got[i].OutputDir = ""
			}
			require.ElementsMatch(t, tt.want, got)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
