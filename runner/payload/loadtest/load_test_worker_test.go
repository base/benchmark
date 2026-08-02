package loadtest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		gasLimit:         150_000_000,
		blockTime:        2 * time.Second,
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
		"mempool_target_blocks: 3",
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
		"block_gas_limit:",
		"ready_file:",
		"start_file:",
		"started_file:",
	} {
		require.NotContains(t, output, oldValue)
	}
}

func TestBuildConfigRemovesTargetGPSAndPreservesExplicitMempoolMultiplier(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "load-test.yaml")
	err := os.WriteFile(configPath, []byte(`
transaction_submission_rpcs:
  - "http://standalone-submitter.invalid"
query_rpc: "http://standalone-query.invalid"
flashblocks_ws: "ws://standalone-flashblocks.invalid"
target_gps: 123
mempool_target_blocks: 5
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

	require.Contains(t, output, "target_gps: null")
	require.Contains(t, output, "mempool_target_blocks: 5")
	require.Contains(t, output, "duration: \"60s\"")
}

func TestBuildConfigAppliesLoadTestConfigOverrides(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "load-test.yaml")
	err := os.WriteFile(configPath, []byte(`
transaction_submission_rpcs:
  - "http://standalone-submitter.invalid"
query_rpc: "http://standalone-query.invalid"
flashblocks_ws: "ws://standalone-flashblocks.invalid"
target_gps: 123
seed: 654789
duration: "60s"
transactions:
  - weight: 100
    type: transfer
`), 0644)
	require.NoError(t, err)

	worker := &loadTestPayloadWorker{
		flashblocksURL:   "ws://benchmark-flashblocks.example",
		gasLimit:         150_000_000,
		blockTime:        2 * time.Second,
		elRPCURL:         "http://sequencer.example",
		sourceConfigPath: configPath,
		configOverrides: map[string]interface{}{
			"seed":                  654_790,
			"fresh_recipient_ratio": 1.0,
		},
	}

	config, err := worker.buildConfig()
	require.NoError(t, err)

	encoded, err := yaml.Marshal(config)
	require.NoError(t, err)
	output := string(encoded)

	require.Contains(t, output, "target_gps: null")
	require.Contains(t, output, "seed: 654790")
	require.Contains(t, output, "fresh_recipient_ratio: 1")
	require.NotContains(t, output, "seed: 654789")
}

func TestSetupStartsProcessAndFirstSendTxsCompletesHandshake(t *testing.T) {
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

	helper := writeHelper(t, `
control=$2
ready="$control/ready"
start="$control/start"
started="$control/started"
finished="$control/finished"
touch "$ready"
while [ ! -e "$start" ]; do sleep 0.01; done
touch "$started"
touch "$finished"
trap 'exit 0' INT TERM
while :; do sleep 1; done
`)
	worker := &loadTestPayloadWorker{
		log:              log.New(),
		loadTestBin:      helper,
		elRPCURL:         "http://sequencer.example",
		sourceConfigPath: configPath,
		done:             make(chan struct{}),
	}
	t.Cleanup(func() { require.NoError(t, worker.Stop(context.Background())) })

	require.NoError(t, worker.Setup(context.Background()))
	require.NotEmpty(t, worker.renderedConfigPath)
	require.NotNil(t, worker.cmd)
	require.FileExists(t, filepath.Join(worker.controlDir, "ready"))
	require.NoFileExists(t, filepath.Join(worker.controlDir, "start"))

	count, err := worker.SendTxs(context.Background(), 0)
	require.NoError(t, err)
	require.Zero(t, count)
	require.FileExists(t, filepath.Join(worker.controlDir, "start"))
	require.FileExists(t, filepath.Join(worker.controlDir, "started"))
	select {
	case <-worker.MeasurementDone():
	case <-time.After(time.Second):
		t.Fatal("measurement completion was not signaled")
	}

	// The barrier is one-shot; subsequent calls return immediately.
	_, err = worker.SendTxs(context.Background(), 0)
	require.NoError(t, err)
}

func TestSetupFailsWhenProcessExitsBeforeReady(t *testing.T) {
	worker := testWorker(t, writeHelper(t, `exit 7`))
	err := worker.Setup(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "before becoming ready")
	require.NoError(t, worker.Stop(context.Background()))
}

func TestSetupHonorsContextCancellation(t *testing.T) {
	worker := testWorker(t, writeHelper(t, `while :; do sleep 1; done`))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := worker.Setup(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, worker.Stop(context.Background()))
}

func TestSendTxsHonorsContextWhileWaitingForStarted(t *testing.T) {
	worker := testWorker(t, writeHelper(t, `
control=$2
ready="$control/ready"
touch "$ready"
trap 'exit 0' INT TERM
while :; do sleep 1; done
`))
	require.NoError(t, worker.Setup(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := worker.SendTxs(ctx, 0)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.FileExists(t, filepath.Join(worker.controlDir, "start"))
	require.NoError(t, worker.Stop(context.Background()))
}

func TestCleanExitBeforeFinishedFailsMeasurement(t *testing.T) {
	worker := testWorker(t, writeHelper(t, `
control=$2
touch "$control/ready"
while [ ! -e "$control/start" ]; do sleep 0.01; done
touch "$control/started"
sleep 0.1
exit 0
`))
	require.NoError(t, worker.Setup(context.Background()))
	_, err := worker.SendTxs(context.Background(), 0)
	require.NoError(t, err)

	select {
	case <-worker.MeasurementDone():
	case <-time.After(time.Second):
		t.Fatal("measurement failure was not signaled")
	}
	require.ErrorContains(t, worker.Err(), "exited before measurement completed")
	require.NoError(t, worker.Stop(context.Background()))
}

func testWorker(t *testing.T, binary string) *loadTestPayloadWorker {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "load-test.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("transactions: []\n"), 0644))
	return &loadTestPayloadWorker{
		log: log.New(), loadTestBin: binary, elRPCURL: "http://sequencer.example",
		sourceConfigPath: configPath, done: make(chan struct{}),
	}
}

func writeHelper(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "helper.sh")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\nset -eu\n"+body), 0755))
	return path
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
