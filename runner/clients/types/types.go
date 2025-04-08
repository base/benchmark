package types

import (
	"context"
	"io"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/ethclient"
)

type RuntimeConfig struct {
	Stdout io.WriteCloser
	Stderr io.WriteCloser
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
}
