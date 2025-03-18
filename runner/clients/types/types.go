package types

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ExecutionClient interface {
	Run(ctx context.Context, chainCfgPath string, jwtSecretPath string, dataDir string) error
	Stop()
	Client() *ethclient.Client
	ClientURL() string // needed for external transaction payload workers
	AuthClient() client.RPC
}
