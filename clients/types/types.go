package types

import (
	gethoptions "github.com/base/base-bench/clients/geth/options"
	rethoptions "github.com/base/base-bench/clients/reth/options"
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
}

type ExecutionClient interface {
	Run(chainCfgPath string, dataDir string) error
	Stop()
	Client() *ethclient.Client
}

type Client uint

const (
	Reth Client = iota
	Geth
)
