package loadtest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBuildConfigOverlaysBenchmarkFieldsAndPreservesLoadTestConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "mainnet-state-weth-usdc-swaps.yaml")
	err := os.WriteFile(configPath, []byte(`
transaction_submission_rpcs:
  - "http://standalone-submitter.invalid"
query_rpc: "http://standalone-query.invalid"
flashblocks_ws: "ws://standalone-flashblocks.invalid"
target_gps: 123
duration: "60s"
chain_id: 8453
sender_count: 250
in_flight_per_sender: 64
batch_size: 20
batch_timeout: "10ms"
seed: 654789
funding_amount: "200000000000000000"
real_token_setup:
  enabled: true
  allow_chain_id_8453: true
  weth: "0x4200000000000000000000000000000000000006"
  weth_amount_per_sender: "50000000000000000"
  pair_token:
    token: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
    amount_per_sender: "10000000"
    acquisition:
      type: uniswap_v3_exact_input
      router: "0x2626664c2603336E57B271c5C0b26F421741e481"
      fee: 500
      amount_in: "10000000000000000"
      min_amount_out: "0"
transactions:
  - weight: 50
    type: uniswap_v3
    router: "0x2626664c2603336E57B271c5C0b26F421741e481"
    token_in: "0x4200000000000000000000000000000000000006"
    token_out: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
    fee: 500
    min_amount: "10000000000000"
    max_amount: "100000000000000"
    reverse_min_amount: "100000"
    reverse_max_amount: "1000000"
  - weight: 50
    type: aerodrome_cl
    router: "0xBE6D8f0d05cC4be24d5167a3eF062215bE6D18a5"
    token_in: "0x4200000000000000000000000000000000000006"
    token_out: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
    tick_spacing: 100
    min_amount: "10000000000000"
    max_amount: "100000000000000"
    reverse_min_amount: "100000"
    reverse_max_amount: "1000000"
`), 0644)
	require.NoError(t, err)

	worker := &loadTestPayloadWorker{
		flashblocksURL:   "ws://benchmark-flashblocks.example",
		targetGPS:        75_000_000,
		elRPCURL:         "http://sequencer.example",
		sourceConfigPath: configPath,
	}

	config, err := worker.buildConfig()
	require.NoError(t, err)

	encoded, err := yaml.Marshal(config)
	require.NoError(t, err)
	output := string(encoded)

	for _, want := range []string{
		"transaction_submission_rpcs:\n    - http://sequencer.example",
		"query_rpc: http://sequencer.example",
		"flashblocks_ws: ws://benchmark-flashblocks.example",
		"target_gps: 75000000",
		"duration: \"60s\"",
		"chain_id: 8453",
		"sender_count: 250",
		"in_flight_per_sender: 64",
		"batch_size: 20",
		"batch_timeout: \"10ms\"",
		"seed: 654789",
		"real_token_setup:",
		"allow_chain_id_8453: true",
		"type: uniswap_v3",
		"type: aerodrome_cl",
		"reverse_min_amount: \"100000\"",
	} {
		require.Contains(t, output, want)
	}
	for _, oldValue := range []string{
		"standalone-submitter.invalid",
		"standalone-query.invalid",
		"standalone-flashblocks.invalid",
		"target_gps: 123",
	} {
		require.NotContains(t, output, oldValue)
	}
}

func TestBuildConfigPreservesNativeTargetGPSWhenBenchmarkTargetUnset(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "load-test.yaml")
	err := os.WriteFile(configPath, []byte(`
transaction_submission_rpcs:
  - "http://standalone-submitter.invalid"
query_rpc: "http://standalone-query.invalid"
flashblocks_ws: "ws://standalone-flashblocks.invalid"
target_gps: 123
duration: "60s"
transactions:
  - weight: 100
    type: transfer
`), 0644)
	require.NoError(t, err)

	worker := &loadTestPayloadWorker{
		flashblocksURL:   "ws://benchmark-flashblocks.example",
		elRPCURL:         "http://sequencer.example",
		sourceConfigPath: configPath,
	}

	config, err := worker.buildConfig()
	require.NoError(t, err)

	encoded, err := yaml.Marshal(config)
	require.NoError(t, err)
	output := string(encoded)

	require.Contains(t, output, "target_gps: 123")
	require.Contains(t, output, "duration: \"60s\"")
}

func TestSetupPreparesConfigWithoutStartingProcess(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "load-test.yaml")
	err := os.WriteFile(configPath, []byte(`
transaction_submission_rpcs:
  - "http://standalone-submitter.invalid"
query_rpc: "http://standalone-query.invalid"
duration: "60s"
transactions:
  - weight: 100
    type: transfer
`), 0644)
	require.NoError(t, err)

	worker := &loadTestPayloadWorker{
		log:              log.New(),
		elRPCURL:         "http://sequencer.example",
		sourceConfigPath: configPath,
		done:             make(chan struct{}),
	}
	t.Cleanup(func() {
		if worker.renderedConfigPath != "" {
			require.NoError(t, os.Remove(worker.renderedConfigPath))
		}
	})

	require.NoError(t, worker.Setup(context.Background()))
	require.NotEmpty(t, worker.renderedConfigPath)
	require.Nil(t, worker.cmd)

	select {
	case <-worker.Done():
		t.Fatal("load-test worker should not be done before it starts")
	default:
	}
}

func TestResolveConfigFilePath(t *testing.T) {
	resolved, err := resolveConfigFilePath("/tmp/configs/benchmark.yml", "load-tests/mainnet.yaml")
	require.NoError(t, err)
	require.Equal(t, "/tmp/configs/load-tests/mainnet.yaml", resolved)

	resolved, err = resolveConfigFilePath("/tmp/configs/benchmark.yml", "/var/load-tests/mainnet.yaml")
	require.NoError(t, err)
	require.Equal(t, "/var/load-tests/mainnet.yaml", resolved)

	_, err = resolveConfigFilePath("/tmp/configs/benchmark.yml", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "config_file")
}
