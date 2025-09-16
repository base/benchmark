package geth

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"

	"github.com/base/base-bench/runner/benchmark/portmanager"
	"github.com/base/base-bench/runner/clients/common"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/metrics"
	"github.com/ethereum/go-ethereum/ethclient"
)

// GethClient handles the lifecycle of a geth client.
type GethClient struct {
	logger  log.Logger
	options *config.InternalClientOptions

	client     *ethclient.Client
	clientURL  string
	authClient client.RPC
	process    *exec.Cmd

	ports       portmanager.PortManager
	metricsPort uint64
	rpcPort     uint64
	authRPCPort uint64

	stdout io.WriteCloser
	stderr io.WriteCloser

	metricsCollector metrics.Collector
}

// NewGethClient creates a new client for geth.
func NewGethClient(logger log.Logger, options *config.InternalClientOptions, ports portmanager.PortManager) types.ExecutionClient {
	return &GethClient{
		logger:  logger,
		options: options,
		ports:   ports,
	}
}

func (g *GethClient) MetricsCollector() metrics.Collector {
	return g.metricsCollector
}

// Run runs the geth client with the given runtime config.
func (g *GethClient) Run(ctx context.Context, cfg *types.RuntimeConfig) error {

	if g.stdout != nil {
		_ = g.stdout.Close()
	}

	if g.stderr != nil {
		_ = g.stderr.Close()
	}

	g.stdout = cfg.Stdout
	g.stderr = cfg.Stderr
	args := make([]string, 0)

	// first init geth
	if !g.options.SkipInit {
		args = append(args, "--datadir", g.options.DataDirPath)
		args = append(args, "--state.scheme", "hash")

		args = append(args, "init", g.options.ChainCfgPath)

		cmd := exec.CommandContext(ctx, g.options.GethBin, args...)
		cmd.Stdout = g.stdout
		cmd.Stderr = g.stderr

		err := cmd.Run()
		if err != nil {
			return errors.Wrap(err, "failed to init geth")
		}
	}

	args = make([]string, 0)
	args = append(args, "--datadir", g.options.DataDirPath)
	args = append(args, "--http")

	g.rpcPort = g.ports.AcquirePort("geth", portmanager.ELPortPurpose)
	g.authRPCPort = g.ports.AcquirePort("geth", portmanager.AuthELPortPurpose)
	g.metricsPort = g.ports.AcquirePort("geth", portmanager.ELMetricsPortPurpose)

	// TODO: allocate these dynamically eventually
	args = append(args, "--http.port", fmt.Sprintf("%d", g.rpcPort))
	args = append(args, "--authrpc.port", fmt.Sprintf("%d", g.authRPCPort))
	args = append(args, "--metrics")
	args = append(args, "--metrics.addr", "localhost")
	args = append(args, "--metrics.port", fmt.Sprintf("%d", g.metricsPort))

	// Set mempool size to 100x default
	args = append(args, "--txpool.globalslots", "10000000")
	args = append(args, "--txpool.globalqueue", "10000000")
	args = append(args, "--txpool.accountslots", "1000000")
	args = append(args, "--txpool.accountqueue", "1000000")
	args = append(args, "--maxpeers", "0")
	args = append(args, "--nodiscover")
	args = append(args, "--rpc.txfeecap", "20")
	args = append(args, "--syncmode", "full")
	args = append(args, "--http.api", "eth,net,web3,miner,debug")
	args = append(args, "--gcmode", "archive")
	args = append(args, "--authrpc.jwtsecret", g.options.JWTSecretPath)

	// TODO: make this configurable
	args = append(args, "--verbosity", "3")

	minerNewPayloadTimeout := time.Second * 2
	args = append(args, "--miner.newpayload-timeout", minerNewPayloadTimeout.String())

	jwtSecretStr, err := os.ReadFile(g.options.JWTSecretPath)
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

	g.logger.Debug("starting geth", "args", strings.Join(args, " "))

	g.process = exec.Command(g.options.GethBin, args...)
	g.process.Stdout = g.stdout
	g.process.Stderr = g.stderr
	err = g.process.Start()
	if err != nil {
		return err
	}

	g.clientURL = fmt.Sprintf("http://127.0.0.1:%d", g.rpcPort)
	rpcClient, err := rpc.DialOptions(ctx, g.clientURL, rpc.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	}))
	if err != nil {
		return errors.Wrap(err, "failed to dial rpc")
	}

	g.client = ethclient.NewClient(rpcClient)
	g.metricsCollector = newMetricsCollector(g.logger, g.client, int(g.metricsPort))

	err = common.WaitForRPC(ctx, g.client)
	if err != nil {
		return errors.Wrap(err, "geth rpc failed to start")
	}

	l2Node, err := client.NewRPC(ctx, g.logger, fmt.Sprintf("http://127.0.0.1:%d", g.authRPCPort), client.WithGethRPCOptions(rpc.WithHTTPAuth(node.NewJWTAuth(jwtSecret))), client.WithCallTimeout(30*time.Second))
	if err != nil {
		return err
	}

	g.authClient = l2Node

	return nil
}

// Stop stops the geth client.
func (g *GethClient) Stop() {
	if g.process == nil || g.process.Process == nil {
		return
	}
	err := g.process.Process.Signal(os.Interrupt)
	if err != nil {
		g.logger.Error("failed to stop geth", "err", err)
	}

	g.process.WaitDelay = 5 * time.Second

	err = g.process.Wait()
	if err != nil {
		g.logger.Error("failed to wait for geth", "err", err)
	}

	_ = g.stdout.Close()
	_ = g.stderr.Close()

	g.ports.ReleasePort(g.rpcPort)
	g.ports.ReleasePort(g.authRPCPort)
	g.ports.ReleasePort(g.metricsPort)

	g.stdout = nil
	g.stderr = nil
	g.process = nil
}

// Client returns the ethclient client.
func (g *GethClient) Client() *ethclient.Client {
	return g.client
}

// ClientURL returns the raw client URL for transaction generators.
func (g *GethClient) ClientURL() string {
	return g.clientURL
}

// AuthClient returns the auth client used for CL communication.
func (g *GethClient) AuthClient() client.RPC {
	return g.authClient
}

func (r *GethClient) MetricsPort() int {
	return int(r.metricsPort)
}

// GetVersion returns the version of the Geth client
func (g *GethClient) GetVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, g.options.GethBin, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "failed to get geth version")
	}

	// Parse version from output - geth version output usually has format like:
	// Geth
	// Version: 1.13.5-stable
	// ...
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Version: ") {
			return strings.TrimPrefix(line, "Version: "), nil
		}
	}

	// Fallback: return first non-empty line if version format is different
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != "Geth" {
			return line, nil
		}
	}

	return "unknown", nil
}

// SetHead resets the blockchain to a specific block using debug.setHead
func (g *GethClient) SetHead(ctx context.Context, blockNumber uint64) error {
	if g.client == nil {
		return errors.New("client not initialized")
	}

	// Convert block number to hex string
	blockHex := fmt.Sprintf("0x%x", blockNumber)

	// Call debug.setHead via RPC
	var result interface{}
	err := g.client.Client().CallContext(ctx, &result, "debug_setHead", blockHex)
	if err != nil {
		return errors.Wrap(err, "failed to call debug_setHead")
	}

	g.logger.Info("Successfully reset blockchain head", "blockNumber", blockNumber, "blockHex", blockHex)
	return nil
}
