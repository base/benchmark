package rbuilder

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"

	"github.com/base/base-bench/runner/benchmark/portmanager"
	"github.com/base/base-bench/runner/clients/geth"
	"github.com/base/base-bench/runner/clients/reth"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/metrics"
)

// RbuilderClient handles the rbuilder (flashblocks) setup.
// Supports two modes:
// 1. Simple mode (rbuilder only): For testing - just rbuilder standalone
// 2. Dual-builder mode: Production architecture with:
//   - Fallback builder (geth/reth): produces final 2s canonical blocks
//   - Rbuilder (primary): produces flashblocks every 200ms
//   - Rollup-boost (optional): coordinates between the two builders
type RbuilderClient struct {
	logger  log.Logger
	options *config.InternalClientOptions
	ports   portmanager.PortManager

	// Simple mode: just rbuilder
	rbuilderClient types.ExecutionClient

	// Dual-builder mode: fallback + rbuilder + optional rollup-boost
	fallbackClient     types.ExecutionClient
	rollupBoostProcess *exec.Cmd
	rollupBoostPort    uint64

	// Client connections (either to rbuilder or rollup-boost)
	client     *ethclient.Client
	clientURL  string
	authClient client.RPC

	stdout io.WriteCloser
	stderr io.WriteCloser

	metricsCollector metrics.Collector

	// Mode tracking
	isDualBuilderMode bool
}

// NewRbuilderClient creates a new rbuilder client.
// Mode is determined by options.RbuilderOptions.FallbackClient:
// - Empty: Simple mode (rbuilder standalone)
// - Set: Dual-builder mode (fallback + rbuilder + optional rollup-boost)
func NewRbuilderClient(logger log.Logger, options *config.InternalClientOptions, ports portmanager.PortManager) types.ExecutionClient {
	return &RbuilderClient{
		logger:  logger,
		options: options,
		ports:   ports,
	}
}

// Run starts the rbuilder setup.
func (r *RbuilderClient) Run(ctx context.Context, cfg *types.RuntimeConfig) error {
	r.stdout = cfg.Stdout
	r.stderr = cfg.Stderr

	// Check if we should run in dual-builder mode
	r.isDualBuilderMode = r.options.RbuilderOptions.FallbackClient != ""

	if r.isDualBuilderMode {
		r.logger.Info("Starting rbuilder in dual-builder mode",
			"fallback", r.options.RbuilderOptions.FallbackClient,
			"rollup_boost", r.options.RbuilderOptions.RollupBoostBin != "")
		return r.runDualBuilderMode(ctx, cfg)
	}

	r.logger.Info("Starting rbuilder in simple mode (standalone)")
	return r.runSimpleMode(ctx, cfg)
}

// runSimpleMode runs rbuilder as a standalone client (original implementation).
func (r *RbuilderClient) runSimpleMode(ctx context.Context, cfg *types.RuntimeConfig) error {
	// Create rbuilder client
	r.rbuilderClient = reth.NewRethClientWithBin(r.logger, r.options, r.ports, r.options.RbuilderBin)

	// Configure flashblocks via environment variables
	cfg2 := *cfg
	cfg2.Env = map[string]string{
		"ENABLE_FLASHBLOCKS": "true",
	}

	// Set flashblock interval if specified
	if cfg.Params.FlashblockInterval > 0 {
		cfg2.Env["FLASHBLOCK_BLOCK_TIME"] = fmt.Sprintf("%d", cfg.Params.FlashblockInterval)
		r.logger.Info("Configuring flashblock interval",
			"interval_ms", cfg.Params.FlashblockInterval,
			"flashblocks_per_block", int(cfg.Params.BlockTime.Milliseconds())/cfg.Params.FlashblockInterval)
	} else {
		// Default to 200ms (Base production default - 10 flashblocks per 2s block)
		cfg2.Env["FLASHBLOCK_BLOCK_TIME"] = "200"
	}

	if err := r.rbuilderClient.Run(ctx, &cfg2); err != nil {
		return errors.Wrap(err, "failed to start rbuilder")
	}

	// Set up client connections
	r.client = r.rbuilderClient.Client()
	r.clientURL = r.rbuilderClient.ClientURL()
	r.authClient = r.rbuilderClient.AuthClient()

	// Set up metrics
	r.metricsCollector = newMetricsCollector(r.logger, r.rbuilderClient.Client(), int(r.rbuilderClient.MetricsPort()))
	if r.metricsCollector == nil {
		return errors.New("failed to create metrics collector")
	}

	return nil
}

// runDualBuilderMode runs the production dual-builder setup.
func (r *RbuilderClient) runDualBuilderMode(ctx context.Context, cfg *types.RuntimeConfig) error {
	// In dual-builder mode, we need separate data directories for each builder
	// to avoid database locking conflicts

	// Step 1: Start fallback builder (produces final 2s blocks)
	fallbackType := r.options.RbuilderOptions.FallbackClient
	r.logger.Info("Starting fallback builder", "type", fallbackType)

	// Create separate log file for fallback builder to avoid file descriptor conflicts
	fallbackLogPath := fmt.Sprintf("%s-fallback.log", r.options.TestDirPath)
	fallbackLogFile, err := os.OpenFile(fallbackLogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to create fallback log file")
	}

	// Create fallback builder with original data directory
	fallbackOptions := *r.options
	fallbackCfg := *cfg
	fallbackCfg.Stdout = fallbackLogFile
	fallbackCfg.Stderr = fallbackLogFile

	if fallbackType == "geth" {
		r.fallbackClient = geth.NewGethClient(r.logger.New("component", "fallback-geth"), &fallbackOptions, r.ports)
	} else {
		r.fallbackClient = reth.NewRethClient(r.logger.New("component", "fallback-reth"), &fallbackOptions, r.ports)
	}

	if err := r.fallbackClient.Run(ctx, &fallbackCfg); err != nil {
		fallbackLogFile.Close()
		return errors.Wrap(err, "failed to start fallback builder")
	}

	r.logger.Info("Fallback builder started", "type", fallbackType)

	// Step 2: Start rbuilder (produces flashblocks every 200ms)
	// Create separate data directory and log file for rbuilder
	r.logger.Info("Starting rbuilder (primary flashblock builder)")

	// Create a fresh data directory for rbuilder
	// Rbuilder only needs to build blocks, not maintain full historical state
	rbuilderDataDir := fmt.Sprintf("%s-rbuilder", r.options.DataDirPath)
	if err := os.MkdirAll(rbuilderDataDir, 0755); err != nil {
		r.fallbackClient.Stop()
		fallbackLogFile.Close()
		return errors.Wrap(err, "failed to create rbuilder data directory")
	}

	// Create separate log file for rbuilder
	rbuilderLogPath := fmt.Sprintf("%s-rbuilder.log", r.options.TestDirPath)
	rbuilderLogFile, err := os.OpenFile(rbuilderLogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		r.fallbackClient.Stop()
		fallbackLogFile.Close()
		return errors.Wrap(err, "failed to create rbuilder log file")
	}

	rbuilderOptions := *r.options
	rbuilderOptions.DataDirPath = rbuilderDataDir

	r.rbuilderClient = reth.NewRethClientWithBin(
		r.logger.New("component", "rbuilder"),
		&rbuilderOptions,
		r.ports,
		r.options.RbuilderBin,
	)

	rbuilderCfg := *cfg
	rbuilderCfg.Stdout = rbuilderLogFile
	rbuilderCfg.Stderr = rbuilderLogFile

	// Disable p2p networking on rbuilder to avoid port conflicts with fallback builder
	// Rbuilder is just a builder, not a full node, so it doesn't need p2p
	rbuilderCfg.Args = append(rbuilderCfg.Args, "--disable-discovery")
	rbuilderCfg.Args = append(rbuilderCfg.Args, "--port", "0") // Disable p2p listener

	// Configure flashblocks via environment variables (not CLI flags)
	// These env vars are used by op-rbuilder based on Base production config
	rbuilderCfg.Env = map[string]string{
		"ENABLE_FLASHBLOCKS": "true",
	}

	// Set flashblock interval if specified
	if cfg.Params.FlashblockInterval > 0 {
		rbuilderCfg.Env["FLASHBLOCK_BLOCK_TIME"] = fmt.Sprintf("%d", cfg.Params.FlashblockInterval)
		r.logger.Info("Configuring flashblock interval",
			"interval_ms", cfg.Params.FlashblockInterval,
			"flashblocks_per_block", int(cfg.Params.BlockTime.Milliseconds())/cfg.Params.FlashblockInterval)
	} else {
		// Default to 200ms (Base production default - 10 flashblocks per 2s block)
		rbuilderCfg.Env["FLASHBLOCK_BLOCK_TIME"] = "200"
	}

	if err := r.rbuilderClient.Run(ctx, &rbuilderCfg); err != nil {
		r.fallbackClient.Stop()
		fallbackLogFile.Close()
		rbuilderLogFile.Close()
		return errors.Wrap(err, "failed to start rbuilder")
	}

	r.logger.Info("Rbuilder started successfully")

	// Step 3: Optionally start rollup-boost coordinator
	if r.options.RbuilderOptions.RollupBoostBin != "" {
		if err := r.startRollupBoost(ctx); err != nil {
			r.rbuilderClient.Stop()
			r.fallbackClient.Stop()
			return errors.Wrap(err, "failed to start rollup-boost")
		}
	} else {
		r.logger.Warn("No rollup-boost binary - connecting directly to rbuilder (not production setup)")
		// Connect directly to rbuilder
		r.client = r.rbuilderClient.Client()
		r.clientURL = r.rbuilderClient.ClientURL()
		r.authClient = r.rbuilderClient.AuthClient()
	}

	// Set up metrics collector for both builders
	r.metricsCollector = newDualBuilderMetricsCollector(
		r.logger,
		r.fallbackClient.Client(),
		r.rbuilderClient.Client(),
		int(r.fallbackClient.MetricsPort()),
		int(r.rbuilderClient.MetricsPort()),
	)
	r.logger.Info("Setup metrics collector for both builders")

	return nil
}

// startRollupBoost starts the rollup-boost coordinator.
func (r *RbuilderClient) startRollupBoost(ctx context.Context) error {
	r.rollupBoostPort = r.ports.AcquirePort("rollup-boost", portmanager.ELPortPurpose)

	// Read JWT secret
	jwtSecretStr, err := os.ReadFile(r.options.JWTSecretPath)
	if err != nil {
		return errors.Wrap(err, "failed to read jwt secret")
	}

	jwtSecretBytes, err := hex.DecodeString(string(jwtSecretStr))
	if err != nil {
		return err
	}

	if len(jwtSecretBytes) != 32 {
		return errors.New("jwt secret must be 32 bytes")
	}

	jwtSecret := [32]byte{}
	copy(jwtSecret[:], jwtSecretBytes[:])

	// Rollup-boost configuration
	fallbackAuthURL := r.fallbackClient.AuthURL()
	rbuilderAuthURL := r.rbuilderClient.AuthURL()

	r.logger.Info("Starting rollup-boost",
		"port", r.rollupBoostPort,
		"fallback_auth", fallbackAuthURL,
		"rbuilder_auth", rbuilderAuthURL)

	// Create separate log file for rollup-boost
	rollupBoostLogPath := fmt.Sprintf("%s-rollup-boost.log", r.options.TestDirPath)
	rollupBoostLogFile, err := os.OpenFile(rollupBoostLogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to create rollup-boost log file")
	}

	// Rollup-boost CLI arguments
	args := []string{
		"--l2-url", fallbackAuthURL,
		"--l2-jwt-path", r.options.JWTSecretPath,
		"--builder-url", rbuilderAuthURL,
		"--builder-jwt-path", r.options.JWTSecretPath,
		"--rpc-port", fmt.Sprintf("%d", r.rollupBoostPort),
		"--execution-mode", "enabled",
		"--metrics",
		"--log-format", "json",
		"--log-level", "debug",
		// Health check configuration for local testing
		// Prevents rollup-boost from shutting down due to old genesis timestamps
		"--health-check-interval", "999999999", // Very long interval for testing
		"--max-unsafe-interval", "999999999", // Allow very old blocks in testing
	}

	r.rollupBoostProcess = exec.CommandContext(ctx, r.options.RbuilderOptions.RollupBoostBin, args...)
	r.rollupBoostProcess.Stdout = rollupBoostLogFile
	r.rollupBoostProcess.Stderr = rollupBoostLogFile

	if err := r.rollupBoostProcess.Start(); err != nil {
		rollupBoostLogFile.Close()
		return errors.Wrap(err, "failed to start rollup-boost process")
	}

	// Wait longer for rollup-boost to initialize and connect to both builders
	r.logger.Info("Waiting for rollup-boost to initialize...")
	time.Sleep(5 * time.Second)

	// Connect to rollup-boost
	r.clientURL = r.rbuilderClient.ClientURL()
	rpcClient, err := rpc.DialOptions(ctx, r.clientURL, rpc.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	}))
	if err != nil {
		return errors.Wrap(err, "failed to dial rollup-boost rpc")
	}

	r.client = ethclient.NewClient(rpcClient)

	// Create auth client
	authRPC, err := client.NewRPC(
		ctx,
		r.logger,
		r.clientURL,
		client.WithGethRPCOptions(rpc.WithHTTPAuth(node.NewJWTAuth(jwtSecret))),
		client.WithCallTimeout(30*time.Second),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create auth client for rollup-boost")
	}

	r.authClient = authRPC
	r.logger.Info("Rollup-boost started successfully")
	return nil
}

func (r *RbuilderClient) MetricsCollector() metrics.Collector {
	return r.metricsCollector
}

// Stop stops all rbuilder components.
func (r *RbuilderClient) Stop() {
	r.logger.Info("Stopping rbuilder setup", "mode", map[bool]string{true: "dual-builder", false: "simple"}[r.isDualBuilderMode])

	// Stop rollup-boost first (if running)
	if r.rollupBoostProcess != nil && r.rollupBoostProcess.Process != nil {
		r.logger.Info("Stopping rollup-boost")
		if err := r.rollupBoostProcess.Process.Signal(os.Interrupt); err != nil {
			r.logger.Error("failed to stop rollup-boost", "err", err)
		}
		r.rollupBoostProcess.Wait()
		r.ports.ReleasePort(r.rollupBoostPort)
	}

	// Stop rbuilder
	if r.rbuilderClient != nil {
		r.logger.Info("Stopping rbuilder")
		r.rbuilderClient.Stop()
	}

	// Stop fallback builder (if dual-builder mode)
	if r.fallbackClient != nil {
		r.logger.Info("Stopping fallback builder")
		r.fallbackClient.Stop()
	}

	if r.stdout != nil {
		_ = r.stdout.Close()
	}
	if r.stderr != nil {
		_ = r.stderr.Close()
	}
}

// Client returns the ethclient (connects through rollup-boost, rbuilder, or fallback).
func (r *RbuilderClient) Client() *ethclient.Client {
	return r.client
}

// ClientURL returns the URL for the client.
func (r *RbuilderClient) ClientURL() string {
	return r.clientURL
}

// AuthURL returns the auth RPC URL.
func (r *RbuilderClient) AuthURL() string {
	if r.rollupBoostPort > 0 {
		// In rollup-boost mode, return rollup-boost's URL
		return fmt.Sprintf("http://127.0.0.1:%d", r.rollupBoostPort)
	}
	// Otherwise return rbuilder's auth URL
	if r.rbuilderClient != nil {
		return r.rbuilderClient.AuthURL()
	}
	return ""
}

// AuthClient returns the auth client for engine API communication.
func (r *RbuilderClient) AuthClient() client.RPC {
	return r.authClient
}

// MetricsPort returns the metrics port.
func (r *RbuilderClient) MetricsPort() int {
	if r.rollupBoostPort > 0 {
		return int(r.rollupBoostPort)
	}
	return r.rbuilderClient.MetricsPort()
}

// GetVersion returns version information.
func (r *RbuilderClient) GetVersion(ctx context.Context) (string, error) {
	// If clients haven't been initialized yet (before Run is called), create a temporary client
	if r.rbuilderClient == nil {
		tempClient := reth.NewRethClientWithBin(r.logger, r.options, r.ports, r.options.RbuilderBin)
		return tempClient.GetVersion(ctx)
	}

	if !r.isDualBuilderMode {
		return r.rbuilderClient.GetVersion(ctx)
	}

	fallbackVersion, _ := r.fallbackClient.GetVersion(ctx)
	rbuilderVersion, _ := r.rbuilderClient.GetVersion(ctx)
	return fmt.Sprintf("flashblocks[fallback:%s,rbuilder:%s]", fallbackVersion, rbuilderVersion), nil
}

// SetHead resets the blockchain head.
func (r *RbuilderClient) SetHead(ctx context.Context, blockNumber uint64) error {
	if !r.isDualBuilderMode {
		return r.rbuilderClient.SetHead(ctx, blockNumber)
	}

	// Reset head on fallback builder (produces final blocks)
	if err := r.fallbackClient.SetHead(ctx, blockNumber); err != nil {
		return errors.Wrap(err, "failed to set head on fallback builder")
	}

	// Also reset on rbuilder (best effort)
	if err := r.rbuilderClient.SetHead(ctx, blockNumber); err != nil {
		r.logger.Warn("Failed to set head on rbuilder", "err", err)
	}

	return nil
}
