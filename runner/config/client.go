package config

import (
	"github.com/urfave/cli/v2"

	"github.com/base/base-bench/runner/benchmark/portmanager"
	gethoptions "github.com/base/base-bench/runner/clients/geth/options"
	rbuilderoptions "github.com/base/base-bench/runner/clients/rbuilder/options"
	rethoptions "github.com/base/base-bench/runner/clients/reth/options"
	"github.com/base/base-bench/runner/flags"
)

// ClientOptions is the common options object that gets passed to execution clients.
type ClientOptions struct {
	CommonOptions
	rethoptions.RethOptions
	gethoptions.GethOptions
	rbuilderoptions.RbuilderOptions
	PortOverrides PortOverrides
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

type PortOverrides map[string]map[portmanager.PortPurpose]uint64

func (o PortOverrides) GetOverride(nodeType string, purpose portmanager.PortPurpose) *uint64 {
	if overrides, ok := o[nodeType]; ok {
		if port, ok := overrides[purpose]; ok {
			return &port
		}
	}
	return nil
}

// ReadClientOptions reads any client options from the CLI context, but certain params may also be
// filled in by test params.
func ReadClientOptions(ctx *cli.Context) ClientOptions {
	// TODO: allow overriding ports via flags

	options := ClientOptions{
		PortOverrides: make(PortOverrides),
		RethOptions: rethoptions.RethOptions{
			RethBin: ctx.String(flags.RethBin),
		},
		GethOptions: gethoptions.GethOptions{
			GethBin: ctx.String(flags.GethBin),
		},
		RbuilderOptions: rbuilderoptions.RbuilderOptions{
			RbuilderBin: ctx.String(flags.RbuilderBin),
		},
	}

	return options
}

// CommonOptions are common client configuration options.
type CommonOptions struct{}
