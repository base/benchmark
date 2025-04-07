package benchmark_test

import (
	"testing"
	"time"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/clients"
	"github.com/stretchr/testify/require"
)

func TestNewMatrixFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  benchmark.Matrix
		want    benchmark.ParamsMatrix
		wantErr bool
	}{
		{
			name: "basic config with single value",
			config: benchmark.Matrix{
				Variables: []benchmark.Param{
					{
						ParamType: benchmark.ParamTypeTxWorkload,
						Value:     stringPtr("simple"),
					},
				},
			},
			want: benchmark.ParamsMatrix{
				{
					NodeType:           clients.Geth,
					TransactionPayload: []benchmark.TransactionPayload{{Type: "simple"}},
					GasLimit:           benchmark.DefaultParams.GasLimit,
					BlockTime:          time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "config with multiple values",
			config: benchmark.Matrix{
				Variables: []benchmark.Param{
					{
						ParamType: benchmark.ParamTypeTxWorkload,
						Values:    &[]string{"simple", "complex"},
					},
					{
						ParamType: benchmark.ParamTypeNode,
						Values:    &[]string{"geth", "reth"},
					},
				},
			},
			want: benchmark.ParamsMatrix{
				{
					NodeType:           clients.Geth,
					GasLimit:           benchmark.DefaultParams.GasLimit,
					TransactionPayload: []benchmark.TransactionPayload{{Type: "simple"}, {Type: "complex"}},
					BlockTime:          time.Second,
				},
				{
					NodeType:           clients.Reth,
					GasLimit:           benchmark.DefaultParams.GasLimit,
					TransactionPayload: []benchmark.TransactionPayload{{Type: "simple"}, {Type: "complex"}},
					BlockTime:          time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "config with multiple values 2x2",
			config: benchmark.Matrix{
				Variables: []benchmark.Param{
					{
						ParamType: benchmark.ParamTypeTxWorkload,
						Values:    &[]string{"simple", "complex"},
					},
					{
						ParamType: benchmark.ParamTypeNode,
						Values:    &[]string{"geth", "reth"},
					},
					{
						ParamType: benchmark.ParamTypeEnv,
						Values:    &[]string{"TEST_ENV=0", "TEST_ENV=1"},
					},
				},
			},
			want: benchmark.ParamsMatrix{
				{
					NodeType:           clients.Geth,
					TransactionPayload: []benchmark.TransactionPayload{{Type: "simple"}, {Type: "complex"}},
					GasLimit:           benchmark.DefaultParams.GasLimit,
					BlockTime:          time.Second,
					Env:                map[string]string{"TEST_ENV": "0"},
				},
				{
					NodeType:           clients.Reth,
					TransactionPayload: []benchmark.TransactionPayload{{Type: "simple"}, {Type: "complex"}},
					GasLimit:           benchmark.DefaultParams.GasLimit,
					BlockTime:          time.Second,
					Env:                map[string]string{"TEST_ENV": "0"},
				},
				{
					NodeType:           clients.Geth,
					TransactionPayload: []benchmark.TransactionPayload{{Type: "simple"}, {Type: "complex"}},
					GasLimit:           benchmark.DefaultParams.GasLimit,
					BlockTime:          time.Second,
					Env:                map[string]string{"TEST_ENV": "1"},
				},
				{
					NodeType:           clients.Reth,
					TransactionPayload: []benchmark.TransactionPayload{{Type: "simple"}, {Type: "complex"}},
					GasLimit:           benchmark.DefaultParams.GasLimit,
					BlockTime:          time.Second,
					Env:                map[string]string{"TEST_ENV": "1"},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate param type",
			config: benchmark.Matrix{
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
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing transaction payload",
			config: benchmark.Matrix{
				Variables: []benchmark.Param{
					{
						ParamType: benchmark.ParamTypeNode,
						Value:     stringPtr("geth"),
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := benchmark.NewParamsMatrixFromConfig(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tt.want, got)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
