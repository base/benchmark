package geth

import (
	"context"
	"encoding/hex"
	"os"
	"os/exec"
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
	"github.com/base/base-bench/runner/logger"
	"github.com/ethereum/go-ethereum/ethclient"
)

// GethClient handles the lifecycle of a geth client.
type GethClient struct {
	logger  log.Logger
	options *config.ClientOptions

	client     *ethclient.Client
	clientURL  string
	authClient client.RPC
	process    *exec.Cmd

	stdout *logger.LogWriter
	stderr *logger.LogWriter
}

// NewGethClient creates a new client for geth.
func NewGethClient(logger log.Logger, options *config.ClientOptions) types.ExecutionClient {
	return &GethClient{
		logger:  logger,
		options: options,
	}
}

// Run runs the geth client with the given runtime config.
func (g *GethClient) Run(ctx context.Context, chainCfgPath string, jwtSecretPath string, dataDir string) error {

	if g.stdout != nil {
		_ = g.stdout.Close()
	}

	if g.stderr != nil {
		_ = g.stderr.Close()
	}

	g.stdout = logger.NewLogWriter(g.logger)
	g.stderr = logger.NewLogWriter(g.logger)

	// first init geth
	args := make([]string, 0)
	args = append(args, "--datadir", dataDir)
	args = append(args, "init", chainCfgPath)

	cmd := exec.CommandContext(ctx, g.options.GethBin, args...)
	cmd.Stdout = g.stdout
	cmd.Stderr = g.stderr

	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "failed to init geth")
	}

	args = make([]string, 0)
	args = append(args, "--datadir", dataDir)
	args = append(args, "--http")

	// TODO: allocate these dynamically eventually
	args = append(args, "--http.port", "8545")
	args = append(args, "--authrpc.port", "8551")

	args = append(args, "--http.api", "eth,net,web3,miner")
	args = append(args, "--authrpc.jwtsecret", jwtSecretPath)

	// TODO: make this configurable
	args = append(args, "--verbosity", "3")

	jwtSecretStr, err := os.ReadFile(jwtSecretPath)
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

	g.clientURL = "http://127.0.0.1:8545"
	rpcClient, err := rpc.Dial(g.clientURL)
	if err != nil {
		return errors.Wrap(err, "failed to dial rpc")
	}

	g.client = ethclient.NewClient(rpcClient)

	err = common.WaitForRPC(ctx, g.client)
	if err != nil {
		return errors.Wrap(err, "geth rpc failed to start")
	}

	l2Node, err := client.NewRPC(ctx, g.logger, "http://127.0.0.1:8551", client.WithGethRPCOptions(rpc.WithHTTPAuth(node.NewJWTAuth(jwtSecret))))
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

	g.stdout.Close()
	g.stderr.Close()

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
