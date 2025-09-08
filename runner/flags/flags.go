package flags

import (
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	RethBin     = "reth-bin"
	RbuilderBin = "rbuilder-bin"
	GethBin     = "geth-bin"
)

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    RethBin,
			Usage:   "Reth binary path",
			Value:   "reth",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "RETH_BIN"),
		},
		&cli.StringFlag{
			Name:    GethBin,
			Usage:   "Geth binary path",
			Value:   "geth",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "GETH_BIN"),
		},
		&cli.StringFlag{
			Name:    RbuilderBin,
			Usage:   "Rbuilder binary path",
			Value:   "rbuilder",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "RBUILDER_BIN"),
		},
	}
}
