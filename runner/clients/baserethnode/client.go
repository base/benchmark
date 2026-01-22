package baserethnode

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
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"

	"github.com/base/base-bench/runner/benchmark/portmanager"
	"github.com/base/base-bench/runner/clients/common"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/metrics"
)

// BaseRethNodeClient handles the lifecycle of a base-reth-node client.
// This client is configured to receive flashblocks via websocket.
type BaseRethNodeClient struct {
	logger  log.Logger
	options *config.InternalClientOptions

	client     *ethclient.Client
	clientURL  string
	authClient client.RPC
	process    *exec.Cmd

	ports       portmanager.PortManager
	metricsPort uint64
	rpcPort     uint64
	p2pPort     uint64
	authRPCPort uint64

	stdout io.WriteCloser
	stderr io.WriteCloser

	metricsCollector metrics.Collector
}

// NewBaseRethNodeClient creates a new client for base-reth-node.
func NewBaseRethNodeClient(logger log.Logger, options *config.InternalClientOptions, ports portmanager.PortManager) types.ExecutionClient {
	return &BaseRethNodeClient{
		logger:  logger,
		options: options,
		ports:   ports,
	}
}

func (r *BaseRethNodeClient) MetricsCollector() metrics.Collector {
	return r.metricsCollector
}

// Run runs the base-reth-node client with the given runtime config.
func (r *BaseRethNodeClient) Run(ctx context.Context, cfg *types.RuntimeConfig) error {
	args := make([]string, 0)
	args = append(args, "node")
	args = append(args, "--color", "never")
	args = append(args, "--chain", r.options.ChainCfgPath)
	args = append(args, "--datadir", r.options.DataDirPath)

	r.rpcPort = r.ports.AcquirePort("base-reth-node", portmanager.ELPortPurpose)
	r.p2pPort = r.ports.AcquirePort("base-reth-node", portmanager.P2PPortPurpose)
	r.authRPCPort = r.ports.AcquirePort("base-reth-node", portmanager.AuthELPortPurpose)
	r.metricsPort = r.ports.AcquirePort("base-reth-node", portmanager.ELMetricsPortPurpose)

	args = append(args, "--http")
	args = append(args, "--http.port", fmt.Sprintf("%d", r.rpcPort))
	args = append(args, "--http.api", "eth,net,web3,miner")
	args = append(args, "--authrpc.port", fmt.Sprintf("%d", r.authRPCPort))
	args = append(args, "--authrpc.jwtsecret", r.options.JWTSecretPath)
	args = append(args, "--metrics", fmt.Sprintf("%d", r.metricsPort))
	args = append(args, "--engine.state-provider-metrics")
	args = append(args, "--disable-discovery")
	args = append(args, "--port", fmt.Sprintf("%d", r.p2pPort))
	args = append(args, "-vvv")

	// increase mempool size
	args = append(args, "--txpool.pending-max-count", "100000000")
	args = append(args, "--txpool.queued-max-count", "100000000")
	args = append(args, "--txpool.pending-max-size", "100")
	args = append(args, "--txpool.queued-max-size", "100")

	args = append(args, "--db.read-transaction-timeout", "0")
	args = append(args, cfg.Args...)

	// Add flashblocks websocket URL if provided
	if cfg.FlashblocksURL != nil && *cfg.FlashblocksURL != "" {
		r.logger.Info("Configuring base-reth-node with flashblocks websocket URL", "url", *cfg.FlashblocksURL)
		args = append(args, "--websocket-url", *cfg.FlashblocksURL)
	}

	// delete datadir/txpool-transactions-backup.rlp if it exists
	txpoolBackupPath := fmt.Sprintf("%s/txpool-transactions-backup.rlp", r.options.DataDirPath)
	if _, err := os.Stat(txpoolBackupPath); err == nil {
		if err := os.Remove(txpoolBackupPath); err != nil {
			return errors.Wrap(err, "failed to remove txpool backup")
		}
	}

	// read jwt secret
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

	if r.stdout != nil {
		_ = r.stdout.Close()
	}

	if r.stderr != nil {
		_ = r.stderr.Close()
	}

	r.stdout = cfg.Stdout
	r.stderr = cfg.Stderr

	r.logger.Debug("starting base-reth-node", "args", strings.Join(args, " "))

	r.process = exec.Command(r.options.BaseRethNodeBin, args...)
	r.process.Stdout = r.stdout
	r.process.Stderr = r.stderr
	err = r.process.Start()
	if err != nil {
		return err
	}

	r.clientURL = fmt.Sprintf("http://127.0.0.1:%d", r.rpcPort)
	rpcClient, err := rpc.DialOptions(ctx, r.clientURL, rpc.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	}))
	if err != nil {
		return errors.Wrap(err, "failed to dial rpc")
	}

	r.client = ethclient.NewClient(rpcClient)
	r.metricsCollector = newMetricsCollector(r.logger, r.client, int(r.metricsPort))

	err = common.WaitForRPC(ctx, r.client)
	if err != nil {
		return errors.Wrap(err, "base-reth-node rpc failed to start")
	}

	l2Node, err := client.NewRPC(ctx, r.logger, fmt.Sprintf("http://127.0.0.1:%d", r.authRPCPort), client.WithGethRPCOptions(rpc.WithHTTPAuth(node.NewJWTAuth(jwtSecret))), client.WithCallTimeout(240*time.Second))
	if err != nil {
		return err
	}

	r.authClient = l2Node

	return nil
}

// Stop stops the base-reth-node client.
func (r *BaseRethNodeClient) Stop() {
	if r.process == nil || r.process.Process == nil {
		return
	}
	err := r.process.Process.Signal(os.Interrupt)
	if err != nil {
		r.logger.Error("failed to stop base-reth-node", "err", err)
	}

	r.process.WaitDelay = 5 * time.Second

	err = r.process.Wait()
	if err != nil {
		r.logger.Error("failed to wait for base-reth-node", "err", err)
	}

	_ = r.stdout.Close()
	_ = r.stderr.Close()

	// Release the ports
	r.ports.ReleasePort(r.rpcPort)
	r.ports.ReleasePort(r.authRPCPort)
	r.ports.ReleasePort(r.metricsPort)
	r.ports.ReleasePort(r.p2pPort)

	r.stdout = nil
	r.stderr = nil
	r.process = nil
}

// Client returns the ethclient client.
func (r *BaseRethNodeClient) Client() *ethclient.Client {
	return r.client
}

// ClientURL returns the raw client URL for transaction generators.
func (r *BaseRethNodeClient) ClientURL() string {
	return r.clientURL
}

// AuthClient returns the auth client used for CL communication.
func (r *BaseRethNodeClient) AuthClient() client.RPC {
	return r.authClient
}

func (r *BaseRethNodeClient) MetricsPort() int {
	return int(r.metricsPort)
}

// GetVersion returns the version of the base-reth-node client
func (r *BaseRethNodeClient) GetVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, r.options.BaseRethNodeBin, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "failed to get base-reth-node version")
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Version:") {
			parts := strings.Split(line, "Version:")
			if len(parts) >= 2 {
				versionPart := strings.TrimSpace(parts[1])
				versionFields := strings.Fields(versionPart)
				if len(versionFields) > 0 {
					return versionFields[0], nil
				}
			}
		}
	}
	return "unknown", nil
}

// SetHead resets the blockchain to a specific block using debug.setHead
func (r *BaseRethNodeClient) SetHead(ctx context.Context, blockNumber uint64) error {
	if r.client == nil {
		return errors.New("client not initialized")
	}

	blockHex := fmt.Sprintf("0x%x", blockNumber)

	var result interface{}
	err := r.client.Client().CallContext(ctx, &result, "debug_setHead", blockHex)
	if err != nil {
		return errors.Wrap(err, "failed to call debug_setHead")
	}

	r.logger.Info("Successfully reset blockchain head", "blockNumber", blockNumber, "blockHex", blockHex)
	return nil
}

// FlashblocksClient returns nil as base-reth-node receives flashblocks but doesn't produce them.
func (r *BaseRethNodeClient) FlashblocksClient() types.FlashblocksClient {
	return nil
}

// SupportsFlashblocks returns true as base-reth-node supports receiving flashblock payloads.
func (r *BaseRethNodeClient) SupportsFlashblocks() bool {
	return true
}
