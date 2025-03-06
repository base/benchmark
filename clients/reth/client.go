package reth

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"

	"github.com/base/base-bench/clients/logger"
	"github.com/base/base-bench/clients/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type RethClient struct {
	logger  log.Logger
	options *types.ClientOptions

	client  *ethclient.Client
	process *exec.Cmd

	stdout *logger.LogWriter
	stderr *logger.LogWriter
}

func NewRethClient(logger log.Logger, options *types.ClientOptions) types.ExecutionClient {
	return &RethClient{
		logger:  logger,
		options: options,
	}
}

func (r *RethClient) Run(chainCfgPath string, dataDir string) error {
	args := make([]string, 0)
	args = append(args, "node")
	args = append(args, "--color", "never")
	args = append(args, "--chain", chainCfgPath)
	args = append(args, "--datadir", dataDir)

	// todo: make this dynamic eventually
	args = append(args, "--http")
	args = append(args, "--http.port", "8545")
	args = append(args, "--http.api", "eth,net,web3")

	if r.stdout != nil {
		_ = r.stdout.Close()
	}

	if r.stderr != nil {
		_ = r.stderr.Close()
	}

	r.stdout = logger.NewLogWriter(r.logger)
	r.stderr = logger.NewLogWriter(r.logger)

	r.logger.Debug("starting reth", "args", strings.Join(args, " "))

	r.process = exec.Command(r.options.RethBin, args...)
	r.process.Stdout = r.stdout
	r.process.Stderr = r.stderr
	err := r.process.Start()
	if err != nil {
		return err
	}

	rpcClient, err := rpc.Dial("http://127.0.0.1:8545")
	if err != nil {
		return errors.Wrap(err, "failed to dial rpc")
	}

	r.client = ethclient.NewClient(rpcClient)
	return nil
}

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

	r.stdout.Close()
	r.stderr.Close()

	r.stdout = nil
	r.stderr = nil
	r.process = nil
}

func (r *RethClient) Client() *ethclient.Client {
	return r.client
}
