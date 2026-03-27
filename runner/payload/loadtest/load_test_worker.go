package loadtest

import (
	"context"
	"crypto/ecdsa"
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
type LoadTestPayloadDefinition struct {
	SenderCount uint64 `yaml:"sender_count"`
	TargetGPS   uint64 `yaml:"target_gps"`
	Duration    string `yaml:"duration"`
}

// loadTestConfig is the YAML config written to a temp file for the load-test binary.
type loadTestConfig struct {
	RPC           string                   `yaml:"rpc"`
	SenderCount   uint64                   `yaml:"sender_count"`
	TargetGPS     uint64                   `yaml:"target_gps"`
	Duration      string                   `yaml:"duration"`
	Seed          uint64                   `yaml:"seed"`
	FundingAmount string                   `yaml:"funding_amount"`
	Transactions  []loadTestTransactionDef `yaml:"transactions"`
}

type loadTestTransactionDef struct {
	Type    string `yaml:"type"`
	Weight  uint64 `yaml:"weight"`
	MaxSize uint64 `yaml:"max_size,omitempty"`
	Target  string `yaml:"target,omitempty"`
}

type loadTestPayloadWorker struct {
	log            log.Logger
	prefundSK      string
	loadTestBin    string
	elRPCURL       string
	gasLimit       uint64
	params         LoadTestPayloadDefinition
	mempool        *mempool.StaticWorkloadMempool
	proxyServer    *proxy.ProxyServer
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

	w := &loadTestPayloadWorker{
		log:         log,
		prefundSK:   hex.EncodeToString(prefundedPrivateKey.D.Bytes()),
		loadTestBin: loadTestBin,
		elRPCURL:    elRPCURL,
		gasLimit:    params.GasLimit,
		params:      definition,
		mempool:     mp,
		proxyServer: ps,
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

	return nil
}

func (w *loadTestPayloadWorker) Stop(ctx context.Context) error {
	w.proxyServer.Stop()

	if w.configFilePath != "" {
		os.Remove(w.configFilePath)
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

// writeConfig generates a temporary YAML config file for the load-test binary
// with the RPC URL pointing to the proxy server.
func (w *loadTestPayloadWorker) writeConfig() (string, error) {
	senderCount := w.params.SenderCount
	if senderCount == 0 {
		senderCount = 10
	}

	targetGPS := w.params.TargetGPS
	if targetGPS == 0 {
		targetGPS = w.gasLimit / 2
	}

	duration := w.params.Duration
	if duration == "" {
		duration = "600s"
	}

	config := loadTestConfig{
		RPC:           fmt.Sprintf("http://localhost:%d", proxyPort),
		SenderCount:   senderCount,
		TargetGPS:     targetGPS,
		Duration:      duration,
		Seed:          12345,
		FundingAmount: "10000000000000000000",
		Transactions: []loadTestTransactionDef{
			{Type: "transfer", Weight: 70},
			{Type: "calldata", Weight: 20, MaxSize: 256},
			{Type: "precompile", Weight: 10, Target: "sha256"},
		},
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal load-test config")
	}

	tmpFile, err := os.CreateTemp("", "load-test-config-*.yaml")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp config file")
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return "", errors.Wrap(err, "failed to write temp config file")
	}

	return tmpFile.Name(), nil
}
