package config

import (
	"github.com/urfave/cli/v2"

	gethoptions "github.com/base/base-bench/runner/clients/geth/options"
	rethoptions "github.com/base/base-bench/runner/clients/reth/options"
	"github.com/base/base-bench/runner/flags"
)

type ClientOptions struct {
	CommonOptions
	rethoptions.RethOptions
	gethoptions.GethOptions
}

func ReadClientOptions(ctx *cli.Context) ClientOptions {
	options := ClientOptions{
		RethOptions: rethoptions.RethOptions{
			RethBin: ctx.String(flags.RethBinFlagName),
		},
		GethOptions: gethoptions.GethOptions{
			GethBin: ctx.String(flags.GethBinFlagName),
		},
	}

	return options
}

type CommonOptions struct {
	JWTSecret string
}
