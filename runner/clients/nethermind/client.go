package nethermind

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

	"github.com/base/base-bench/runner/benchmark/portmanager"
	"github.com/base/base-bench/runner/clients/common"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/metrics"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

// NethermindClient handles the lifecycle of a Nethermind client.
type NethermindClient struct {
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

	binPath          string
	metricsCollector metrics.Collector
}

// NewNethermindClient creates a new client for Nethermind.
func NewNethermindClient(logger log.Logger, options *config.InternalClientOptions, ports portmanager.PortManager) types.ExecutionClient {
	return &NethermindClient{
		logger:  logger,
		options: options,
		ports:   ports,
		binPath: options.NethermindBin,
	}
}

func (n *NethermindClient) MetricsCollector() metrics.Collector {
	return n.metricsCollector
}

// Run runs the Nethermind client with the given runtime config.
func (n *NethermindClient) Run(ctx context.Context, cfg *types.RuntimeConfig) error {
	if n.stdout != nil {
		_ = n.stdout.Close()
	}

	if n.stderr != nil {
		_ = n.stderr.Close()
	}

	n.stdout = cfg.Stdout
	n.stderr = cfg.Stderr

	// Allocate ports
	n.rpcPort = n.ports.AcquirePort("nethermind", portmanager.ELPortPurpose)
	n.authRPCPort = n.ports.AcquirePort("nethermind", portmanager.AuthELPortPurpose)
	n.metricsPort = n.ports.AcquirePort("nethermind", portmanager.ELMetricsPortPurpose)

	// Build command line arguments
	args := make([]string, 0)

	// Basic configuration
	args = append(args, "--data-dir", n.options.DataDirPath)
	args = append(args, "--hive-genesisfilepath", n.options.ChainCfgPath)

	// Enable JSON-RPC
	args = append(args, "--jsonrpc-enabled", "true")
	args = append(args, "--jsonrpc-host", "127.0.0.1")
	args = append(args, "--jsonrpc-port", fmt.Sprintf("%d", n.rpcPort))
	args = append(args, "--jsonrpc-enabledmodules", "eth,net,web3,debug")

	// Enable Engine API
	args = append(args, "--jsonrpc-enginehost", "127.0.0.1")
	args = append(args, "--jsonrpc-engineport", fmt.Sprintf("%d", n.authRPCPort))
	args = append(args, "--jsonrpc-jwtsecretfile", n.options.JWTSecretPath)

	// Enable metrics
	args = append(args, "--metrics-enabled", "true")
	args = append(args, "--metrics-exposehost", "127.0.0.1")
	args = append(args, "--metrics-exposeport", fmt.Sprintf("%d", n.metricsPort))

	// Network configuration - disable P2P for benchmarking
	args = append(args, "--network-maxactivepeers", "0")
	args = append(args, "--init-discoveryenabled", "false")

	// Logging
	args = append(args, "--log", "info")

	// Read and validate JWT secret
	jwtSecretStr, err := os.ReadFile(n.options.JWTSecretPath)
	if err != nil {
		return errors.Wrap(err, "failed to read jwt secret")
	}

	jwtSecretBytes, err := hex.DecodeString(strings.TrimSpace(string(jwtSecretStr)))
	if err != nil {
		return errors.Wrap(err, "failed to decode jwt secret")
	}

	if len(jwtSecretBytes) != 32 {
		return errors.New("jwt secret must be 32 bytes")
	}

	jwtSecret := [32]byte{}
	copy(jwtSecret[:], jwtSecretBytes[:])

	n.logger.Debug("starting nethermind", "args", strings.Join(args, " "))

	// Start Nethermind process
	n.process = exec.Command(n.binPath, args...)
	n.process.Stdout = n.stdout
	n.process.Stderr = n.stderr
	err = n.process.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start nethermind")
	}

	// Set up RPC client
	n.clientURL = fmt.Sprintf("http://127.0.0.1:%d", n.rpcPort)
	rpcClient, err := rpc.DialOptions(ctx, n.clientURL, rpc.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	}))
	if err != nil {
		return errors.Wrap(err, "failed to dial rpc")
	}

	n.client = ethclient.NewClient(rpcClient)
	n.metricsCollector = newMetricsCollector(n.logger, n.client, int(n.metricsPort))

	// Wait for RPC to be ready
	err = common.WaitForRPC(ctx, n.client)
	if err != nil {
		return errors.Wrap(err, "nethermind rpc failed to start")
	}

	// Set up Engine API client
	authClient, err := client.NewRPC(ctx, n.logger, fmt.Sprintf("http://127.0.0.1:%d", n.authRPCPort),
		client.WithGethRPCOptions(rpc.WithHTTPAuth(node.NewJWTAuth(jwtSecret))),
		client.WithCallTimeout(30*time.Second))
	if err != nil {
		return errors.Wrap(err, "failed to create auth client")
	}

	n.authClient = authClient

	return nil
}

func (n *NethermindClient) Stop() {
	if n.process == nil || n.process.Process == nil {
		return
	}
	err := n.process.Process.Signal(os.Interrupt)
	if err != nil {
		n.logger.Error("failed to stop nethermind", "err", err)
	}

	n.process.WaitDelay = 5 * time.Second

	err = n.process.Wait()
	if err != nil {
		n.logger.Error("failed to wait for nethermind", "err", err)
	}

	_ = n.stdout.Close()
	_ = n.stderr.Close()

	n.ports.ReleasePort(n.rpcPort)
	n.ports.ReleasePort(n.authRPCPort)
	n.ports.ReleasePort(n.metricsPort)

	n.stdout = nil
	n.stderr = nil
	n.process = nil
}

func (n *NethermindClient) Client() *ethclient.Client {
	return n.client
}

func (n *NethermindClient) ClientURL() string {
	return n.clientURL
}

func (n *NethermindClient) AuthClient() client.RPC {
	return n.authClient
}

func (n *NethermindClient) MetricsPort() int {
	return int(n.metricsPort)
}
