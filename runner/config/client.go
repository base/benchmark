package config

import (
	"github.com/urfave/cli/v2"

	gethoptions "github.com/base/base-bench/runner/clients/geth/options"
	rethoptions "github.com/base/base-bench/runner/clients/reth/options"
	"github.com/base/base-bench/runner/flags"
)

// ClientOptions is the common options object that gets passed to execution clients.
type ClientOptions struct {
	CommonOptions
	rethoptions.RethOptions
	gethoptions.GethOptions
}

// InternalProofProgramOptions are options that the user can set when running the proof program.
type ProofProgramOptions struct {
	Enabled *bool
	Version *string
}

// InternalProofProgramOptions are options that are used internally by the proof program runner.
type InternalProofProgramOptions struct {
	ProofProgramOptions

	BlobsDir string
}

// InternalClientOptions are options that are set internally by the runner.
type InternalClientOptions struct {
	ClientOptions

	JWTSecretPath string
	ChainCfgPath  string
	DataDirPath   string
	TestDirPath   string
	JWTSecret     string
	MetricsPath   string
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
type CommonOptions struct{}
