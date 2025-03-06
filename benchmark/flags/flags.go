package flags

import (
	"github.com/base/base-bench/clients/types"
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const EnvVarPrefix = "BASE_BENCH"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

const (
	ConfigFlagName  = "config"
	RootDirFlagName = "root-dir"
)

var (
	ConfigFlag = &cli.StringFlag{
		Name:     "config",
		Usage:    "Config Path",
		EnvVars:  prefixEnvVars("CONFIG"),
		Required: true,
	}

	RootDirFlag = &cli.StringFlag{
		Name:     "root-dir",
		Usage:    "Root Directory",
		EnvVars:  prefixEnvVars("ROOT_DIR"),
		Required: true,
	}
)

// Flags contains the list of configuration options available to the binary.
var Flags = []cli.Flag{
	ConfigFlag,
	RootDirFlag,
}

func init() {
	Flags = append(Flags, oplog.CLIFlags(EnvVarPrefix)...)
	Flags = append(Flags, types.CLIFlags(EnvVarPrefix)...)
}
