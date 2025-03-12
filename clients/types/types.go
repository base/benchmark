package types

import (
	"context"

	gethoptions "github.com/base/base-bench/clients/geth/options"
	rethoptions "github.com/base/base-bench/clients/reth/options"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

type ClientOptions struct {
	CommonOptions
	rethoptions.RethOptions
	gethoptions.GethOptions
}

func ReadClientOptions(ctx *cli.Context) ClientOptions {
	options := ClientOptions{
		RethOptions: rethoptions.RethOptions{
			RethBin: ctx.String(RethBinFlagName),
		},
		GethOptions: gethoptions.GethOptions{
			GethBin: ctx.String(GethBinFlagName),
		},
	}

	return options
}

type CommonOptions struct {
	JWTSecret string
}

type ExecutionClient interface {
	Run(ctx context.Context, chainCfgPath string, jwtSecretPath string, dataDir string) error
	Stop()
	Client() *ethclient.Client // TODO: switch to *client.RPC
	ClientURL() string         // needed for external transaction payload workers
	AuthClient() client.RPC
}

type Client uint

const (
	Reth Client = iota
	Geth
)
