package loadtest

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// LoadTestPayloadDefinition is the YAML payload params for the load-test type.
// The load-test workload itself lives in a native base-load-tester config file;
// benchmark mode overlays the RPC and block fields it must control.
type LoadTestPayloadDefinition struct {
	ConfigFile string `yaml:"config_file"`
	Network    string `yaml:"network"`
}

type loadTestPayloadWorker struct {
	log                log.Logger
	prefundSK          string
	loadTestBin        string
	elRPCURL           string
	flashblocksURL     string
	gasLimit           uint64
	blockTime          time.Duration
	params             LoadTestPayloadDefinition
	configOverrides    map[string]interface{}
	mempool            *mempool.StaticWorkloadMempool
	cmd                *exec.Cmd
	done               chan struct{}
	measurementDone    chan struct{}
	processOnce        sync.Once
	armOnce            sync.Once
	finishOnce         sync.Once
	shutdownOnce       sync.Once
	waitErrMu          sync.Mutex
	waitErr            error
	armErr             error
	sourceConfigPath   string
	renderedConfigPath string
	controlDir         string
	outputPath         string
}

// NewLoadTestPayloadWorker creates a worker that runs the base-load-tester binary
// as an external transaction generator against the benchmark node's RPC.
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

	sourceConfigPath, err := resolveConfigFilePath(cfg.ConfigPath(), definition.ConfigFile)
	if err != nil {
		return nil, err
	}

	w := &loadTestPayloadWorker{
		log:              log,
		prefundSK:        hex.EncodeToString(prefundedPrivateKey.D.Bytes()),
		loadTestBin:      cfg.LoadTestBinary(),
		elRPCURL:         elRPCURL,
		flashblocksURL:   flashblocksURL,
		gasLimit:         params.GasLimit,
		blockTime:        params.BlockTime,
		params:           definition,
		configOverrides:  params.LoadTestConfigOverrides,
		mempool:          mp,
		done:             make(chan struct{}),
		measurementDone:  make(chan struct{}),
		sourceConfigPath: sourceConfigPath,
		outputPath:       outputPath,
	}

	return w, nil
}

func (w *loadTestPayloadWorker) Mempool() mempool.FakeMempool {
	return w.mempool
}

func (w *loadTestPayloadWorker) Setup(ctx context.Context) error {
	configPath, err := w.writeConfig()
	if err != nil {
		return errors.Wrap(err, "failed to write load-test config")
	}
	w.renderedConfigPath = configPath

	if err := w.start(ctx); err != nil {
		return err
	}
	readyFile := filepath.Join(w.controlDir, "ready")
	w.log.Info("Waiting for load test readiness", "ready_file", readyFile)
	return w.waitForFile(ctx, readyFile, "load test exited before becoming ready")
}

func (w *loadTestPayloadWorker) start(ctx context.Context) error {
	w.processOnce.Do(func() {
		if w.renderedConfigPath == "" {
			w.finish(errors.New("load-test config has not been prepared"))
			return
		}

		w.log.Info("Starting load test", "binary", w.loadTestBin, "config", w.renderedConfigPath)

		cmd := exec.CommandContext(
			ctx,
			w.loadTestBin,
			"--separate-setup", w.controlDir,
			"--block-gas-limit", strconv.FormatUint(w.gasLimit, 10),
			w.renderedConfigPath,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout
		cmd.Env = append(os.Environ(), fmt.Sprintf("FUNDER_KEY=%s", w.prefundSK))
		if w.outputPath != "" {
			if err := os.MkdirAll(filepath.Dir(w.outputPath), 0755); err != nil {
				w.finish(errors.Wrap(err, "failed to create load-test output directory"))
				return
			}
			cmd.Env = append(cmd.Env, fmt.Sprintf("LOAD_TEST_OUTPUT=%s", w.outputPath))
		}

		if err := cmd.Start(); err != nil {
			w.finish(errors.Wrap(err, "failed to start load test binary"))
			return
		}
		w.cmd = cmd
		if w.measurementDone == nil {
			w.measurementDone = make(chan struct{})
		}
		go func() {
			w.finish(cmd.Wait())
		}()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
			defer cancel()
			err := w.waitForFile(
				ctx,
				filepath.Join(w.controlDir, "finished"),
				"load test exited before measurement completed",
			)
			if err != nil {
				w.waitErrMu.Lock()
				if w.waitErr == nil {
					w.waitErr = err
				}
				w.waitErrMu.Unlock()
			}
			close(w.measurementDone)
		}()
	})

	return w.Err()
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
	return w.done
}

func (w *loadTestPayloadWorker) MeasurementDone() <-chan struct{} {
	return w.measurementDone
}

func (w *loadTestPayloadWorker) Err() error {
	w.waitErrMu.Lock()
	defer w.waitErrMu.Unlock()
	return w.waitErr
}

func (w *loadTestPayloadWorker) finish(err error) {
	w.finishOnce.Do(func() {
		w.waitErrMu.Lock()
		if w.waitErr == nil {
			w.waitErr = err
		}
		w.waitErrMu.Unlock()
		close(w.done)
	})
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

	if w.renderedConfigPath != "" {
		if err := os.Remove(w.renderedConfigPath); err != nil {
			w.log.Warn("failed to remove load-test config", "path", w.renderedConfigPath, "err", err)
		}
	}
	if w.controlDir != "" {
		if err := os.RemoveAll(w.controlDir); err != nil {
			w.log.Warn("failed to remove load-test control directory", "path", w.controlDir, "err", err)
		}
	}

	return nil
}

func (w *loadTestPayloadWorker) SendTxs(ctx context.Context, _ int) (int, error) {
	w.armOnce.Do(func() {
		startFile := filepath.Join(w.controlDir, "start")
		if err := publishFile(startFile); err != nil {
			w.armErr = errors.Wrap(err, "failed to arm load test")
			return
		}
		startedFile := filepath.Join(w.controlDir, "started")
		w.log.Info("Armed load test; waiting for start acknowledgement", "started_file", startedFile)
		w.armErr = w.waitForFile(ctx, startedFile, "load test exited before acknowledging start")
	})
	return 0, w.armErr
}

func (w *loadTestPayloadWorker) waitForFile(ctx context.Context, path, exitMessage string) error {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return errors.Wrap(err, "failed to inspect load-test handshake file")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.Done():
			if err := w.Err(); err != nil {
				return errors.Wrap(err, exitMessage)
			}
			return errors.New(exitMessage)
		case <-ticker.C:
		}
	}
}

func publishFile(path string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".load-test-handshake-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func resolveConfigFilePath(benchmarkConfigPath string, loadTestConfigPath string) (string, error) {
	if loadTestConfigPath == "" {
		return "", errors.New("load-test payload requires config_file")
	}
	if filepath.IsAbs(loadTestConfigPath) {
		return loadTestConfigPath, nil
	}
	return filepath.Join(filepath.Dir(benchmarkConfigPath), loadTestConfigPath), nil
}

func (w *loadTestPayloadWorker) buildConfig() (*yaml.Node, error) {
	data, err := os.ReadFile(w.sourceConfigPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read load-test config file")
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, errors.Wrap(err, "failed to parse load-test config file")
	}

	config, err := mappingRoot(&doc)
	if err != nil {
		return nil, err
	}

	setMappingValue(config, "transaction_submission_rpcs", stringSequenceNode(w.elRPCURL))
	setMappingValue(config, "query_rpc", stringNode(w.elRPCURL))

	flashblocksURL := w.flashblocksURL
	if flashblocksURL == "" {
		flashblocksURL = "ws://localhost:7111"
	}
	setMappingValue(config, "flashblocks_ws", stringNode(flashblocksURL))
	if mappingUintValue(config, "mempool_target_blocks") == 0 {
		setMappingValue(config, "mempool_target_blocks", uintNode(3))
	}
	for key, value := range w.configOverrides {
		node, err := nodeFromValue(value)
		if err != nil {
			return nil, fmt.Errorf("invalid load-test config override %q: %w", key, err)
		}
		setMappingValue(config, key, node)
	}
	setMappingValue(config, "target_gps", nullNode())

	return config, nil
}

func mappingRoot(doc *yaml.Node) (*yaml.Node, error) {
	root := doc
	if doc.Kind == yaml.DocumentNode {
		if len(doc.Content) == 0 {
			return nil, errors.New("load-test config file is empty")
		}
		root = doc.Content[0]
	}

	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("load-test config file must be a YAML mapping, got kind %d", root.Kind)
	}
	return root, nil
}

func setMappingValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = value
			return
		}
	}
	mapping.Content = append(mapping.Content, stringNode(key), value)
}

func mappingUintValue(mapping *yaml.Node, key string) uint64 {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			value, _ := strconv.ParseUint(mapping.Content[i+1].Value, 10, 64)
			return value
		}
	}
	return 0
}

func stringNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func uintNode(value uint64) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatUint(value, 10)}
}

func nodeFromValue(value interface{}) (*yaml.Node, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		return nil, errors.New("empty YAML value")
	}
	return doc.Content[0], nil
}

func nullNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
}

func stringSequenceNode(values ...string) *yaml.Node {
	node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, value := range values {
		node.Content = append(node.Content, stringNode(value))
	}
	return node
}

// writeConfig generates a temporary YAML config file for the load-test binary
// with benchmark-controlled RPC, block, handshake, and report fields.
func (w *loadTestPayloadWorker) writeConfig() (string, error) {
	tmpFile, err := os.CreateTemp("", "load-test-config-*.yaml")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp config file")
	}
	configPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		return "", errors.Wrap(err, "failed to close temp config file")
	}
	w.controlDir, err = os.MkdirTemp("", "load-test-control-*")
	if err != nil {
		_ = os.Remove(configPath)
		return "", errors.Wrap(err, "failed to create load-test control directory")
	}

	config, err := w.buildConfig()
	if err != nil {
		_ = os.Remove(configPath)
		return "", err
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal load-test config")
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		_ = os.Remove(configPath)
		return "", errors.Wrap(err, "failed to write temp config file")
	}

	w.log.Info("Generated load-test config",
		"source_config", w.sourceConfigPath,
		"gas_limit", w.gasLimit,
		"block_time", w.blockTime,
	)

	return configPath, nil
}
