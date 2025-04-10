package config

import (
	gethoptions "github.com/base/base-bench/runner/clients/geth/options"
	rethoptions "github.com/base/base-bench/runner/clients/reth/options"
	"github.com/base/base-bench/runner/flags"
	"github.com/urfave/cli/v2"
)

// ClientOptions is the common options object that gets passed to execution clients.
type ClientOptions struct {
	CommonOptions
	rethoptions.RethOptions
	gethoptions.GethOptions
}

// ReadClientOptions reads any client options from the CLI context, but certain params may also be
// filled in by test params.
func ReadClientOptions(ctx *cli.Context) ClientOptions {
	options := ClientOptions{
		RethOptions: rethoptions.RethOptions{
			RethBin:         ctx.String(flags.RethBin),
			RethHttpPort:    ctx.Int(flags.RethHttpPort),
			RethAuthRpcPort: ctx.Int(flags.RethAuthRpcPort),
			RethMetricsPort: ctx.Int(flags.RethMetricsPort),
		},
		GethOptions: gethoptions.GethOptions{
			GethBin:         ctx.String(flags.GethBin),
			GethHttpPort:    ctx.Int(flags.GethHttpPort),
			GethAuthRpcPort: ctx.Int(flags.GethAuthRpcPort),
			GethMetricsPort: ctx.Int(flags.GethMetricsPort),
		},
	}

	return options
}

// CommonOptions are common client configuration options.
type CommonOptions struct {
	JWTSecret string
}
