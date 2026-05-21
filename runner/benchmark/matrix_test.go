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

func TestNewTestPlanFromConfigRoles(t *testing.T) {
	config := &benchmark.BenchmarkConfig{Name: "test"}
	definition := benchmark.TestDefinition{
		Roles: []benchmark.BenchmarkRole{benchmark.BenchmarkRoleSequencer},
		Variables: []benchmark.Param{
			{
				ParamType: "payload",
				Value:     "simple",
			},
		},
	}

	plan, err := benchmark.NewTestPlanFromConfig(definition, "config.yml", config)
	require.NoError(t, err)
	require.False(t, plan.Mode.RunValidator)

	metadata := benchmark.RunGroupFromTestPlans([]benchmark.TestPlan{*plan}, nil)
	require.Len(t, metadata.Runs, 1)
	require.Equal(t, "sequencer", metadata.Runs[0].TestConfig["Roles"])
}

func TestNewTestPlanFromConfigDefaultsToBothRoles(t *testing.T) {
	config := &benchmark.BenchmarkConfig{Name: "test"}

	definition := benchmark.TestDefinition{
		Variables: []benchmark.Param{
			{
				ParamType: "payload",
				Value:     "simple",
			},
		},
	}

	plan, err := benchmark.NewTestPlanFromConfig(definition, "config.yml", config)
	require.NoError(t, err)
	require.True(t, plan.Mode.RunValidator)

	metadata := benchmark.RunGroupFromTestPlans([]benchmark.TestPlan{*plan}, nil)
	require.Len(t, metadata.Runs, 1)
	require.NotContains(t, metadata.Runs[0].TestConfig, "Roles")
}

func TestNewTestPlanFromConfigRejectsInvalidRoles(t *testing.T) {
	tests := []struct {
		name  string
		roles []benchmark.BenchmarkRole
	}{
		{
			name:  "unknown role",
			roles: []benchmark.BenchmarkRole{"other"},
		},
		{
			name:  "duplicate role",
			roles: []benchmark.BenchmarkRole{benchmark.BenchmarkRoleSequencer, benchmark.BenchmarkRoleSequencer},
		},
		{
			name:  "validator without sequencer",
			roles: []benchmark.BenchmarkRole{benchmark.BenchmarkRoleValidator},
		},
	}

	config := &benchmark.BenchmarkConfig{Name: "test"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition := benchmark.TestDefinition{
				Roles: tt.roles,
				Variables: []benchmark.Param{
					{
						ParamType: "payload",
						Value:     "simple",
					},
				},
			}

			_, err := benchmark.NewTestPlanFromConfig(definition, "config.yml", config)
			require.Error(t, err)
		})
	}
}

func TestNewTestPlanFromConfigRejectsProofProgramWithoutValidator(t *testing.T) {
	config := &benchmark.BenchmarkConfig{Name: "test"}
	definition := benchmark.TestDefinition{
		Roles: []benchmark.BenchmarkRole{benchmark.BenchmarkRoleSequencer},
		ProofProgram: &benchmark.ProofProgramOptions{
			Enabled: boolPtr(true),
		},
		Variables: []benchmark.Param{
			{
				ParamType: "payload",
				Value:     "simple",
			},
		},
	}

	_, err := benchmark.NewTestPlanFromConfig(definition, "config.yml", config)
	require.ErrorContains(t, err, "proof_program requires the validator benchmark role")
}

func TestNewTestPlanFromConfigRejectsValidatorThresholdsWithoutValidator(t *testing.T) {
	config := &benchmark.BenchmarkConfig{Name: "test"}
	definition := benchmark.TestDefinition{
		Roles: []benchmark.BenchmarkRole{benchmark.BenchmarkRoleSequencer},
		Metrics: &benchmark.ThresholdConfig{
			Error: map[string]float64{
				"validator/latency/new_payload": 1e9,
			},
		},
		Variables: []benchmark.Param{
			{
				ParamType: "payload",
				Value:     "simple",
			},
		},
	}

	_, err := benchmark.NewTestPlanFromConfig(definition, "config.yml", config)
	require.ErrorContains(t, err, `error threshold "validator/latency/new_payload" requires the validator benchmark role`)
}

func TestNewTestPlanFromConfigAllowsSequencerThresholdsWithoutValidator(t *testing.T) {
	config := &benchmark.BenchmarkConfig{Name: "test"}
	definition := benchmark.TestDefinition{
		Roles: []benchmark.BenchmarkRole{benchmark.BenchmarkRoleSequencer},
		Metrics: &benchmark.ThresholdConfig{
			Error: map[string]float64{
				"sequencer/latency/get_payload": 1e9,
			},
		},
		Variables: []benchmark.Param{
			{
				ParamType: "payload",
				Value:     "simple",
			},
		},
	}

	plan, err := benchmark.NewTestPlanFromConfig(definition, "config.yml", config)
	require.NoError(t, err)
	require.False(t, plan.Mode.RunValidator)
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
			{
				ParamType: "consensus_timing",
				Value:     types.ConsensusTimingModeBaseConsensus,
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
	require.Equal(t, types.ConsensusTimingModeBaseConsensus, runs[0].Params.ConsensusTimingMode)
	require.Equal(t, types.ConsensusTimingModeBaseConsensus, runs[1].Params.ConsensusTimingMode)
	require.Equal(t, types.ConsensusTimingModeBaseConsensus, runs[2].Params.ConsensusTimingMode)
}

func TestResolveTestRunsFromMatrixRejectsInvalidConsensusTiming(t *testing.T) {
	config := &benchmark.BenchmarkConfig{Name: "benchmark"}
	definition := benchmark.TestDefinition{
		Variables: []benchmark.Param{
			{
				ParamType: "consensus_timing",
				Value:     "aligned",
			},
		},
	}

	_, err := benchmark.ResolveTestRunsFromMatrix(definition, "benchmark.yml", config)
	require.ErrorContains(t, err, "invalid consensus timing")
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
