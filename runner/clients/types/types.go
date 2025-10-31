package types

import (
	"context"
	"io"

	"github.com/base/base-bench/runner/metrics"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/ethclient"
)

type RuntimeConfig struct {
	Stdout io.WriteCloser
	Stderr io.WriteCloser
	Args   []string
	Env    map[string]string    // Environment variables to set for the client process
	Params benchtypes.RunParams // Benchmark parameters for client-specific configuration
}

// ExecutionClient is an abstraction over the different clients that can be used to run the chain like
// op-reth and op-geth.
type ExecutionClient interface {
	Run(ctx context.Context, config *RuntimeConfig) error
	Stop()
	Client() *ethclient.Client
	ClientURL() string // HTTP RPC URL for external transaction payload workers
	AuthClient() client.RPC
	AuthURL() string // Auth RPC URL (for rollup-boost and other auth connections)
	MetricsPort() int
	MetricsCollector() metrics.Collector
	GetVersion(ctx context.Context) (string, error)
	SetHead(ctx context.Context, blockNumber uint64) error
}
