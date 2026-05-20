package benchmark_test

import (
	"testing"
	"time"

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
						BlockTime: 1 * time.Second,
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
						BlockTime: 1 * time.Second,
					},
				},
				{
					Params: types.RunParams{
						NodeType:  "erigon",
						GasLimit:  benchmark.DefaultParams.GasLimit,
						PayloadID: "simple",
						BlockTime: 1 * time.Second,
					},
				},
				{
					Params: types.RunParams{
						NodeType:  "geth",
						GasLimit:  benchmark.DefaultParams.GasLimit,
						PayloadID: "complex",
						BlockTime: 1 * time.Second,
					},
				},
				{
					Params: types.RunParams{
						NodeType:  "erigon",
						GasLimit:  benchmark.DefaultParams.GasLimit,
						PayloadID: "complex",
						BlockTime: 1 * time.Second,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "config with target gps",
			config: benchmark.TestDefinition{
				Variables: []benchmark.Param{
					{
						ParamType: "payload",
						Value:     "load-test",
					},
					{
						ParamType: "target_gps",
						Value:     200_000_000,
					},
				},
			},
			want: []benchmark.TestRun{
				{
					Params: types.RunParams{
						NodeType:  "geth",
						PayloadID: "load-test",
						GasLimit:  benchmark.DefaultParams.GasLimit,
						TargetGPS: 200_000_000,
						BlockTime: 1 * time.Second,
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

func TestResolveTestRunsFromMatrixExpandsTargetGPSValues(t *testing.T) {
	config := &benchmark.BenchmarkConfig{Name: "snapshot load test"}
	definition := benchmark.TestDefinition{
		Variables: []benchmark.Param{
			{
				ParamType: "payload",
				Value:     "mainnet-snapshot-load-test",
			},
			{
				ParamType: "node_type",
				Value:     "builder",
			},
			{
				ParamType: "gas_limit",
				Value:     1_200_000_000,
			},
			{
				ParamType: "target_gps",
				Values: []interface{}{
					80_000_000,
					400_000_000,
					1_200_000_000,
				},
			},
		},
	}

	runs, err := benchmark.ResolveTestRunsFromMatrix(definition, "snapshot-load-test.yml", config)
	require.NoError(t, err)
	require.Len(t, runs, 3)

	require.Equal(t, uint64(1_200_000_000), runs[0].Params.GasLimit)
	require.Equal(t, uint64(1_200_000_000), runs[1].Params.GasLimit)
	require.Equal(t, uint64(1_200_000_000), runs[2].Params.GasLimit)
	require.Equal(t, uint64(80_000_000), runs[0].Params.TargetGPS)
	require.Equal(t, uint64(400_000_000), runs[1].Params.TargetGPS)
	require.Equal(t, uint64(1_200_000_000), runs[2].Params.TargetGPS)
}

func stringPtr(s string) *string {
	return &s
}
