package types

import (
	"context"
	"io"

	"github.com/base/base-bench/runner/metrics"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/ethclient"
)

type RuntimeConfig struct {
	Stdout io.WriteCloser
	Stderr io.WriteCloser
	Args   []string
}

// ExecutionClient is an abstraction over the different clients that can be used to run the chain like
// op-reth and op-geth.
type ExecutionClient interface {
	Run(ctx context.Context, config *RuntimeConfig) error
	Stop()
	Client() *ethclient.Client
	ClientURL() string // needed for external transaction payload workers
	AuthClient() client.RPC
	MetricsPort() int
	MetricsCollector() metrics.Collector
	GetVersion(ctx context.Context) (string, error)
	SetHead(ctx context.Context, blockNumber uint64) error
}
