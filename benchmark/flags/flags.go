package flags

import (
	"github.com/base/base-bench/runner/flags"
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const EnvVarPrefix = "BASE_BENCH"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

const (
	ConfigFlagName          = "config"
	RootDirFlagName         = "root-dir"
	OutputDirFlagName       = "output-dir"
	TxFuzzBinFlagName       = "tx-fuzz-bin"
	ProxyPortFlagName       = "proxy-port"
	BenchmarkRunIDFlagName  = "benchmark-run-id"
	MachineTypeFlagName     = "machine-type"
	MachineProviderFlagName = "machine-provider"
	MachineRegionFlagName   = "machine-region"
	FileSystemFlagName      = "file-system"
	ParallelTxBatchesFlagName = "parallel-tx-batches"
)

// TxFuzz defaults
const (
	DefaultTxFuzzBin = "../tx-fuzz/cmd/livefuzzer/livefuzzer"
)

var (
	ConfigFlag = &cli.StringFlag{
		Name:     ConfigFlagName,
		Usage:    "Config Path",
		EnvVars:  prefixEnvVars("CONFIG"),
		Required: true,
	}

	RootDirFlag = &cli.StringFlag{
		Name:     RootDirFlagName,
		Usage:    "Root Directory",
		EnvVars:  prefixEnvVars("ROOT_DIR"),
		Required: true,
	}

	OutputDirFlag = &cli.StringFlag{
		Name:     OutputDirFlagName,
		Usage:    "Output Directory",
		EnvVars:  prefixEnvVars("OUTPUT_DIR"),
		Required: true,
	}

	TxFuzzBinFlag = &cli.StringFlag{
		Name:    TxFuzzBinFlagName,
		Usage:   "Transaction Fuzzer binary path",
		Value:   DefaultTxFuzzBin,
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "TX_FUZZ_BIN"),
	}

	ProxyPortFlag = &cli.IntFlag{
		Name:    "proxy-port",
		Usage:   "Proxy port",
		Value:   8546,
		EnvVars: prefixEnvVars("PROXY_PORT"),
	}

	BenchmarkRunIDFlag = &cli.StringFlag{
		Name:    BenchmarkRunIDFlagName,
		Usage:   "Custom benchmark run ID (auto-generated if not provided)",
		EnvVars: prefixEnvVars("BENCHMARK_RUN_ID"),
	}

	MachineTypeFlag = &cli.StringFlag{
		Name:    "machine-type",
		Usage:   "Machine type (e.g., i4i.32xlarge)",
		EnvVars: prefixEnvVars("MACHINE_TYPE"),
	}

	MachineProviderFlag = &cli.StringFlag{
		Name:    "machine-provider",
		Usage:   "Cloud provider (aws or gcp)",
		EnvVars: prefixEnvVars("MACHINE_PROVIDER"),
	}

	MachineRegionFlag = &cli.StringFlag{
		Name:    "machine-region",
		Usage:   "Machine region",
		EnvVars: prefixEnvVars("MACHINE_REGION"),
	}

	FileSystemFlag = &cli.StringFlag{
		Name:    "file-system",
		Usage:   "File system type",
		Value:   "ext4",
		EnvVars: prefixEnvVars("FILE_SYSTEM"),
	}

	ParallelTxBatchesFlag = &cli.IntFlag{
		Name:    ParallelTxBatchesFlagName,
		Usage:   "Number of parallel batches for sending transactions",
		Value:   4,
		EnvVars: prefixEnvVars("PARALLEL_TX_BATCHES"),
	}
)

// Flags contains the list of configuration options available to the binary.
var Flags = []cli.Flag{}

var RunFlags = []cli.Flag{
	ConfigFlag,
	RootDirFlag,
	OutputDirFlag,
	TxFuzzBinFlag,
	ProxyPortFlag,
	BenchmarkRunIDFlag,
	MachineTypeFlag,
	MachineProviderFlag,
	MachineRegionFlag,
	FileSystemFlag,
	ParallelTxBatchesFlag,
}

func init() {
	Flags = append(Flags, oplog.CLIFlags(EnvVarPrefix)...)
	RunFlags = append(RunFlags, flags.CLIFlags(EnvVarPrefix)...)
}
