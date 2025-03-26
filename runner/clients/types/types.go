package types

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ExecutionClient is an abstraction over the different clients that can be used to run the chain like
// op-reth and op-geth.
type ExecutionClient interface {
	Run(ctx context.Context, chainCfgPath string, jwtSecretPath string, dataDir string) error
	Stop()
	Client() *ethclient.Client
	ClientURL() string // needed for external transaction payload workers
	AuthClient() client.RPC
}
