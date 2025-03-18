package flags

import (
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	RethBinFlagName = "reth-bin"
	GethBinFlagName = "geth-bin"
)

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    RethBinFlagName,
			Usage:   "Reth binary path",
			Value:   "reth",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "RETH_BIN"),
		},
		&cli.StringFlag{
			Name:    GethBinFlagName,
			Usage:   "Geth binary path",
			Value:   "geth",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "GETH_BIN"),
		},
	}
}
