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
	"path/filepath"
	"sync"
	"time"

	"github.com/base/base-bench/runner/clients/common/proxy"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

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
	RPC                       string    `yaml:"rpc,omitempty"`
	TransactionSubmissionRPCs []string  `yaml:"transaction_submission_rpcs"`
	QueryRPC                  string    `yaml:"query_rpc"`
	FlashblocksWs             string    `yaml:"flashblocks_ws"`
	SenderCount               uint64    `yaml:"sender_count"`
	TargetGPS                 uint64    `yaml:"target_gps"`
	Duration                  string    `yaml:"duration"`
	Seed                      uint64    `yaml:"seed"`
	FundingAmount             string    `yaml:"funding_amount"`
	Transactions              yaml.Node `yaml:"transactions"`
}

type loadTestPayloadWorker struct {
	log            log.Logger
	prefundSK      string
	loadTestBin    string
	elRPCURL       string
	flashblocksURL string
	gasLimit       uint64
	blockTimeSec   uint64
	params         LoadTestPayloadDefinition
	mempool        *mempool.StaticWorkloadMempool
	proxyServer    *proxy.ProxyServer
	cmd            *exec.Cmd
	done           chan struct{}
	waitErr        error
	waitMu         sync.Mutex
	shutdownOnce   sync.Once
	configFilePath string
	outputPath     string
}

// NewLoadTestPayloadWorker creates a worker that runs the base-load-test binary
// as an external transaction generator, capturing transactions via a proxy server.
func NewLoadTestPayloadWorker(
	log log.Logger,
	elRPCURL string,
	flashblocksURL string,
	params types.RunParams,
	prefundedPrivateKey ecdsa.PrivateKey,
	prefundAmount *big.Int,
	cfg config.Config,
	chainID *big.Int,
	definition LoadTestPayloadDefinition,
	outputPath string,
) (worker.Worker, error) {
	mp := mempool.NewStaticWorkloadMempool(log, chainID)
	ps := proxy.NewProxyServer(elRPCURL, log, cfg.ProxyPort(), mp)

	blockTimeSec := uint64(params.BlockTime.Seconds())
	if blockTimeSec == 0 {
		blockTimeSec = 1
	}

	w := &loadTestPayloadWorker{
		log:            log,
		prefundSK:      hex.EncodeToString(prefundedPrivateKey.D.Bytes()),
		loadTestBin:    cfg.LoadTestBinary(),
		elRPCURL:       elRPCURL,
		flashblocksURL: flashblocksURL,
		gasLimit:       params.GasLimit,
		blockTimeSec:   blockTimeSec,
		params:         definition,
		mempool:        mp,
		proxyServer:    ps,
		outputPath:     outputPath,
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
	if w.outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(w.outputPath), 0755); err != nil {
			return errors.Wrap(err, "failed to create load-test output directory")
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("LOAD_TEST_OUTPUT=%s", w.outputPath))
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start load test binary")
	}
	w.cmd = cmd
	w.done = make(chan struct{})
	go func() {
		err := cmd.Wait()
		w.waitMu.Lock()
		w.waitErr = err
		w.waitMu.Unlock()
		close(w.done)
	}()

	return nil
}

func (w *loadTestPayloadWorker) BeginGracefulShutdown(ctx context.Context) error {
	if w.cmd == nil || w.cmd.Process == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-w.Done():
		return nil
	default:
	}

	var signalErr error
	w.shutdownOnce.Do(func() {
		w.log.Info("Stopping load test process gracefully", "pid", w.cmd.Process.Pid, "output", w.outputPath)
		signalErr = w.cmd.Process.Signal(os.Interrupt)
	})
	if signalErr != nil {
		select {
		case <-w.Done():
			return nil
		default:
		}
	}
	return signalErr
}

func (w *loadTestPayloadWorker) Done() <-chan struct{} {
	if w.done != nil {
		return w.done
	}

	done := make(chan struct{})
	close(done)
	return done
}

func (w *loadTestPayloadWorker) Stop(ctx context.Context) error {
	if w.cmd != nil && w.cmd.Process != nil {
		if err := w.BeginGracefulShutdown(ctx); err != nil {
			w.log.Warn("failed to signal load test process", "err", err)
		}

		select {
		case <-w.Done():
		case <-time.After(10 * time.Second):
			w.log.Warn("load test process did not stop gracefully, killing", "pid", w.cmd.Process.Pid)
			if err := w.cmd.Process.Kill(); err != nil {
				w.log.Warn("failed to kill load test process", "err", err)
			}
			select {
			case <-w.Done():
			case <-time.After(5 * time.Second):
				w.log.Warn("timed out waiting for killed load test process")
			}
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

func (w *loadTestPayloadWorker) SendTxs(ctx context.Context, _ int) (int, error) {
	w.log.Info("Collecting txs from load test")
	pendingTxs := w.proxyServer.DrainPendingTxs()

	w.mempool.AddTransactions(pendingTxs)
	return len(pendingTxs), nil
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

	flashblocksURL := w.flashblocksURL
	if flashblocksURL == "" {
		flashblocksURL = "ws://localhost:7111"
	}

	config := loadTestConfig{
		RPC:                       w.proxyServer.ClientURL(),
		TransactionSubmissionRPCs: []string{w.proxyServer.ClientURL()},
		QueryRPC:                  w.proxyServer.ClientURL(),
		FlashblocksWs:             flashblocksURL,
		SenderCount:               senderCount,
		TargetGPS:                 targetGPS,
		Duration:                  "99999s",
		Seed:                      randomSeed(),
		FundingAmount:             fundingAmount,
		Transactions:              transactions,
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
		_ = tmpFile.Close()
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
