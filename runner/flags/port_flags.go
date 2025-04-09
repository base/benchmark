package flags

import (
	"github.com/base/base-bench/runner/portconfig"
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	GethHTTPPortFlag    = "geth.http.port"
	GethAuthRPCPortFlag = "geth.authrpc.port"
	GethMetricsPortFlag = "geth.metrics.port"

	RethHTTPPortFlag    = "reth.http.port"
	RethAuthRPCPortFlag = "reth.authrpc.port"
	RethMetricsPortFlag = "reth.metrics.port"
)

func PortFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    GethHTTPPortFlag,
			Usage:   "HTTP-RPC server listening port for Geth",
			Value:   portconfig.DefaultGethHTTPPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "GETH_HTTP_PORT"),
		},
		&cli.IntFlag{
			Name:    GethAuthRPCPortFlag,
			Usage:   "Auth-RPC server listening port for Geth",
			Value:   portconfig.DefaultGethAuthRPCPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "GETH_AUTHRPC_PORT"),
		},
		&cli.IntFlag{
			Name:    GethMetricsPortFlag,
			Usage:   "Metrics server listening port for Geth",
			Value:   portconfig.DefaultGethMetricsPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "GETH_METRICS_PORT"),
		},
		&cli.IntFlag{
			Name:    RethHTTPPortFlag,
			Usage:   "HTTP-RPC server listening port for Reth",
			Value:   portconfig.DefaultRethHTTPPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "RETH_HTTP_PORT"),
		},
		&cli.IntFlag{
			Name:    RethAuthRPCPortFlag,
			Usage:   "Auth-RPC server listening port for Reth",
			Value:   portconfig.DefaultRethAuthRPCPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "RETH_AUTHRPC_PORT"),
		},
		&cli.IntFlag{
			Name:    RethMetricsPortFlag,
			Usage:   "Metrics server listening port for Reth",
			Value:   portconfig.DefaultRethMetricsPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "RETH_METRICS_PORT"),
		},
	}
}
