package flags

import (
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	RethBin         = "reth-bin"
	RethHttpPort    = "reth-http-port"
	RethAuthRpcPort = "reth-auth-rpc-port"
	RethMetricsPort = "reth-metrics-port"
	GethBin         = "geth-bin"
	GethHttpPort    = "geth-http-port"
	GethAuthRpcPort = "geth-auth-rpc-port"
	GethMetricsPort = "geth-metrics-port"
)

const (
	// Geth defaults
	DefaultGethHTTPPort    = 8545
	DefaultGethAuthRPCPort = 8551
	DefaultGethMetricsPort = 8080

	// Reth defaults
	DefaultRethHTTPPort    = 9545
	DefaultRethAuthRPCPort = 9551
	DefaultRethMetricsPort = 9080
)

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    RethBin,
			Usage:   "Reth binary path",
			Value:   "reth",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "RETH_BIN"),
		},
		&cli.IntFlag{
			Name:    RethHttpPort,
			Usage:   "Reth HTTP port",
			Value:   DefaultRethHTTPPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "RETH_HTTP_PORT"),
		},
		&cli.IntFlag{
			Name:    RethAuthRpcPort,
			Usage:   "Reth Auth RPC port",
			Value:   DefaultRethAuthRPCPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "RETH_AUTHRPC_PORT"),
		},
		&cli.IntFlag{
			Name:    RethMetricsPort,
			Usage:   "Reth Metrics port",
			Value:   DefaultRethMetricsPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "RETH_METRICS_PORT"),
		},

		&cli.StringFlag{
			Name:    GethBin,
			Usage:   "Geth binary path",
			Value:   "geth",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "GETH_BIN"),
		},
		&cli.IntFlag{
			Name:    GethHttpPort,
			Usage:   "Geth HTTP port",
			Value:   DefaultGethHTTPPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "GETH_HTTP_PORT"),
		},
		&cli.IntFlag{
			Name:    GethAuthRpcPort,
			Usage:   "Geth Auth RPC port",
			Value:   DefaultGethAuthRPCPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "GETH_AUTHRPC_PORT"),
		},
		&cli.IntFlag{
			Name:    GethMetricsPort,
			Usage:   "Geth Metrics port",
			Value:   DefaultGethMetricsPort,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "GETH_METRICS_PORT"),
		},
	}
}
