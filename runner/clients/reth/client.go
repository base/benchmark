package reth

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"

	"github.com/base/base-bench/runner/clients/common"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/ethereum/go-ethereum/ethclient"
)

// RethClient handles the lifecycle of a reth client.
type RethClient struct {
	logger  log.Logger
	options *config.InternalClientOptions

	client     *ethclient.Client
	clientURL  string
	authClient client.RPC
	process    *exec.Cmd

	stdout io.WriteCloser
	stderr io.WriteCloser
}

// NewRethClient creates a new client for reth.
func NewRethClient(logger log.Logger, options *config.InternalClientOptions) types.ExecutionClient {
	return &RethClient{
		logger:  logger,
		options: options,
	}
}

// Run runs the reth client with the given runtime config.
func (r *RethClient) Run(ctx context.Context, cfg *types.RuntimeConfig) error {
	args := make([]string, 0)
	args = append(args, "node")
	args = append(args, "--color", "never")
	args = append(args, "--chain", r.options.ChainCfgPath)
	args = append(args, "--datadir", r.options.DataDirPath)

	// todo: make this dynamic eventually
	args = append(args, "--http")
	args = append(args, "--http.port", strconv.Itoa(r.options.RethHttpPort))
	args = append(args, "--http.api", "eth,net,web3,miner")
	args = append(args, "--authrpc.port", strconv.Itoa(r.options.RethAuthRpcPort))
	args = append(args, "--authrpc.jwtsecret", r.options.JWTSecretPath)
	args = append(args, "--metrics", strconv.Itoa(r.options.RethMetricsPort))
	args = append(args, "-vvv")

	// increase mempool size
	args = append(args, "--txpool.pending-max-count", "100000000")
	args = append(args, "--txpool.queued-max-count", "100000000")
	args = append(args, "--txpool.pending-max-size", "100")
	args = append(args, "--txpool.queued-max-size", "100")

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

	r.logger.Debug("starting reth", "args", strings.Join(args, " "))

	r.process = exec.Command(r.options.RethBin, args...)
	r.process.Stdout = r.stdout
	r.process.Stderr = r.stderr
	err = r.process.Start()
	if err != nil {
		return err
	}

	r.clientURL = fmt.Sprintf("http://127.0.0.1:%d", r.options.RethHttpPort)
	rpcClient, err := rpc.DialOptions(ctx, r.clientURL, rpc.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	}))
	if err != nil {
		return errors.Wrap(err, "failed to dial rpc")
	}

	r.client = ethclient.NewClient(rpcClient)

	err = common.WaitForRPC(ctx, r.client)
	if err != nil {
		return errors.Wrap(err, "geth rpc failed to start")
	}

	l2Node, err := client.NewRPC(ctx, r.logger, fmt.Sprintf("http://127.0.0.1:%d", r.options.RethAuthRpcPort), client.WithGethRPCOptions(rpc.WithHTTPAuth(node.NewJWTAuth(jwtSecret))), client.WithCallTimeout(30*time.Second))
	if err != nil {
		return err
	}

	r.authClient = l2Node

	return nil
}

// Stop stops the reth client.
func (r *RethClient) Stop() {
	if r.process == nil || r.process.Process == nil {
		return
	}
	err := r.process.Process.Signal(os.Interrupt)
	if err != nil {
		r.logger.Error("failed to stop reth", "err", err)
	}

	r.process.WaitDelay = 5 * time.Second

	err = r.process.Wait()
	if err != nil {
		r.logger.Error("failed to wait for reth", "err", err)
	}

	_ = r.stdout.Close()
	_ = r.stderr.Close()

	r.stdout = nil
	r.stderr = nil
	r.process = nil
}

// Client returns the ethclient client.
func (r *RethClient) Client() *ethclient.Client {
	return r.client
}

// ClientURL returns the raw client URL for transaction generators.
func (r *RethClient) ClientURL() string {
	return r.clientURL
}

// AuthClient returns the auth client used for CL communication.
func (r *RethClient) AuthClient() client.RPC {
	return r.authClient
}

func (r *RethClient) MetricsPort() int {
	return r.options.RethMetricsPort
}
