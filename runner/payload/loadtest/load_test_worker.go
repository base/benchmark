package loadtest

import (
	"context"
	"crypto/ecdsa"
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"os/exec"

	"github.com/base/base-bench/runner/clients/common/proxy"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const proxyPort = 8545

// LoadTestPayloadDefinition is the YAML payload params for the load-test type.
// Fields map directly to the Rust base-load-test config format.
// The `transactions` field is passed through as raw YAML to support the full
// Rust config schema (transfer, calldata, precompile, erc20, etc.).
type LoadTestPayloadDefinition struct {
	SenderCount   uint64    `yaml:"sender_count"`
	FundingAmount string    `yaml:"funding_amount"`
	Transactions  yaml.Node `yaml:"transactions"`
}

// loadTestConfig is the YAML config written to a temp file for the load-test binary.
type loadTestConfig struct {
	RPC           string    `yaml:"rpc"`
	SenderCount   uint64    `yaml:"sender_count"`
	TargetGPS     uint64    `yaml:"target_gps"`
	Duration      string    `yaml:"duration"`
	Seed          uint64    `yaml:"seed"`
	FundingAmount string    `yaml:"funding_amount"`
	Transactions  yaml.Node `yaml:"transactions"`
}

type loadTestPayloadWorker struct {
	log            log.Logger
	prefundSK      string
	loadTestBin    string
	elRPCURL       string
	gasLimit       uint64
	blockTimeSec   uint64
	params         LoadTestPayloadDefinition
	mempool        *mempool.StaticWorkloadMempool
	proxyServer    *proxy.ProxyServer
	cmd            *exec.Cmd
	configFilePath string
}

// NewLoadTestPayloadWorker creates a worker that runs the base-load-test binary
// as an external transaction generator, capturing transactions via a proxy server.
func NewLoadTestPayloadWorker(
	log log.Logger,
	elRPCURL string,
	params types.RunParams,
	prefundedPrivateKey ecdsa.PrivateKey,
	prefundAmount *big.Int,
	loadTestBin string,
	chainID *big.Int,
	definition LoadTestPayloadDefinition,
) (worker.Worker, error) {
	mp := mempool.NewStaticWorkloadMempool(log, chainID)
	ps := proxy.NewProxyServer(elRPCURL, log, proxyPort, mp)

	blockTimeSec := uint64(params.BlockTime.Seconds())
	if blockTimeSec == 0 {
		blockTimeSec = 1
	}

	w := &loadTestPayloadWorker{
		log:          log,
		prefundSK:    hex.EncodeToString(prefundedPrivateKey.D.Bytes()),
		loadTestBin:  loadTestBin,
		elRPCURL:     elRPCURL,
		gasLimit:     params.GasLimit,
		blockTimeSec: blockTimeSec,
		params:       definition,
		mempool:      mp,
		proxyServer:  ps,
	}

	return w, nil
}

func (w *loadTestPayloadWorker) Mempool() mempool.FakeMempool {
	return w.mempool
}

func (w *loadTestPayloadWorker) Setup(ctx context.Context) error {
	if err := w.proxyServer.Run(ctx); err != nil {
		return errors.Wrap(err, "failed to run proxy server")
	}

	configPath, err := w.writeConfig()
	if err != nil {
		return errors.Wrap(err, "failed to write load-test config")
	}
	w.configFilePath = configPath

	w.log.Info("Starting load test", "binary", w.loadTestBin, "config", configPath)

	cmd := exec.CommandContext(ctx, w.loadTestBin, configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	cmd.Env = append(os.Environ(), fmt.Sprintf("FUNDER_KEY=%s", w.prefundSK))

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start load test binary")
	}
	w.cmd = cmd

	return nil
}

func (w *loadTestPayloadWorker) Stop(ctx context.Context) error {
	if w.cmd != nil && w.cmd.Process != nil {
		w.log.Info("Stopping load test process", "pid", w.cmd.Process.Pid)
		if err := w.cmd.Process.Kill(); err != nil {
			w.log.Warn("failed to kill load test process", "err", err)
		} else {
			// Reap the process to avoid zombies.
			_, _ = w.cmd.Process.Wait()
		}
	}

	w.proxyServer.Stop()

	if w.configFilePath != "" {
		if err := os.Remove(w.configFilePath); err != nil {
			w.log.Warn("failed to remove load-test config", "path", w.configFilePath, "err", err)
		}
	}

	return nil
}

func (w *loadTestPayloadWorker) SendTxs(ctx context.Context) error {
	w.log.Info("Collecting txs from load test")
	pendingTxs := w.proxyServer.PendingTxs()
	w.proxyServer.ClearPendingTxs()

	w.mempool.AddTransactions(pendingTxs)
	return nil
}

// defaultTransactions returns the default transaction mix as a yaml.Node.
func defaultTransactions() yaml.Node {
	var node yaml.Node
	// Default: 70% transfer, 20% calldata, 10% precompile
	defaultYAML := `
- weight: 70
  type: transfer
- weight: 20
  type: calldata
  max_size: 256
- weight: 10
  type: precompile
  target: sha256
`
	if err := yaml.Unmarshal([]byte(defaultYAML), &node); err != nil {
		panic(fmt.Sprintf("failed to parse default transactions YAML: %v", err))
	}
	// yaml.Unmarshal wraps in a document node; return the inner sequence
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return *node.Content[0]
	}
	return node
}

// randomSeed returns a cryptographically random uint64 seed.
func randomSeed() uint64 {
	var b [8]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		return 42
	}
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

// writeConfig generates a temporary YAML config file for the load-test binary
// with the RPC URL pointing to the proxy server.
func (w *loadTestPayloadWorker) writeConfig() (string, error) {
	senderCount := w.params.SenderCount
	if senderCount == 0 {
		senderCount = 10
	}

	fundingAmount := w.params.FundingAmount
	if fundingAmount == "" {
		fundingAmount = "10000000000000000000"
	}

	// Compute target GPS from gas limit and block time
	targetGPS := w.gasLimit / w.blockTimeSec

	transactions := w.params.Transactions
	if transactions.Kind == 0 {
		transactions = defaultTransactions()
	}

	config := loadTestConfig{
		RPC:           fmt.Sprintf("http://localhost:%d", proxyPort),
		SenderCount:   senderCount,
		TargetGPS:     targetGPS,
		Duration:      "99999s",
		Seed:          randomSeed(),
		FundingAmount: fundingAmount,
		Transactions:  transactions,
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal load-test config")
	}

	tmpFile, err := os.CreateTemp("", "load-test-config-*.yaml")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp config file")
	}

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return "", errors.Wrap(err, "failed to write temp config file")
	}

	if err := tmpFile.Close(); err != nil {
		return "", errors.Wrap(err, "failed to close temp config file")
	}

	w.log.Info("Generated load-test config",
		"sender_count", senderCount,
		"target_gps", targetGPS,
		"gas_limit", w.gasLimit,
		"block_time_sec", w.blockTimeSec,
	)

	return tmpFile.Name(), nil
}
