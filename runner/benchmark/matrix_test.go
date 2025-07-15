package benchmark_test

import (
	"testing"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/network/types"
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
						ParamType: "payload",
						Value:     stringPtr("simple"),
					},
				},
			},
			want: []benchmark.TestRun{
				{
					Params: types.RunParams{
						NodeType:  "geth",
						PayloadID: "simple",
						GasLimit:  benchmark.DefaultParams.GasLimit,
						BlockTime: benchmark.DefaultParams.BlockTime,
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
						ParamType: "payload",
						Values:    []interface{}{"simple", "complex"},
					},
					{
						ParamType: "node_type",
						Values:    []interface{}{"geth", "erigon"},
					},
				},
			},
			want: []benchmark.TestRun{
				{
					Params: types.RunParams{
						NodeType:  "geth",
						GasLimit:  benchmark.DefaultParams.GasLimit,
						PayloadID: "simple",
						BlockTime: benchmark.DefaultParams.BlockTime,
					},
				},
				{
					Params: types.RunParams{
						NodeType:  "erigon",
						GasLimit:  benchmark.DefaultParams.GasLimit,
						PayloadID: "simple",
						BlockTime: benchmark.DefaultParams.BlockTime,
					},
				},
				{
					Params: types.RunParams{
						NodeType:  "geth",
						GasLimit:  benchmark.DefaultParams.GasLimit,
						PayloadID: "complex",
						BlockTime: benchmark.DefaultParams.BlockTime,
					},
				},
				{
					Params: types.RunParams{
						NodeType:  "erigon",
						GasLimit:  benchmark.DefaultParams.GasLimit,
						PayloadID: "complex",
						BlockTime: benchmark.DefaultParams.BlockTime,
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
						ParamType: "payload",
						Value:     stringPtr("simple"),
					},
					{
						ParamType: "payload",
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
						ParamType: "node_type",
						Value:     stringPtr("geth"),
					},
				},
			},
			want:    []benchmark.TestRun{},
			wantErr: true,
		},
	}

	config := &benchmark.BenchmarkConfig{
		Name:        "test",
		Description: stringPtr("test"),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := benchmark.ResolveTestRunsFromMatrix(tt.config, "", config)

			if tt.wantErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}
			// ignore outputDir and id
			for i := range tt.want {
				tt.want[i].OutputDir = ""
				tt.want[i].Params.BenchmarkRunID = ""
				tt.want[i].ID = ""
				tt.want[i].Name = "test"
				tt.want[i].Description = "test"
				tt.want[i].Params.Name = "test"
				tt.want[i].Params.Description = "test"
			}
			for i := range got {
				got[i].OutputDir = ""
				got[i].Params.BenchmarkRunID = ""
				got[i].ID = ""
				got[i].Name = "test"
				got[i].Description = "test"
				got[i].Params.Name = "test"
				got[i].Params.Description = "test"
			}
			require.ElementsMatch(t, tt.want, got)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
