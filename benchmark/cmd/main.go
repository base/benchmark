package main

import (
	"context"
	"fmt"
	"os"

	"github.com/base/base-bench/benchmark/config"
	"github.com/base/base-bench/benchmark/flags"
	"github.com/base/base-bench/runner"
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
)

// autopopulated by the Makefile
var (
	Version   = ""
	GitCommit = ""
	GitDate   = ""
)

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Commands = []*cli.Command{
		{
			Name:        "run",
			Flags:       cliapp.ProtectFlags(flags.RunFlags),
			Action:      Main(Version),
			Usage:       "run benchmark",
			Description: "Runs benchmarks according to the specified config.",
		},
	}
	app.Flags = flags.Flags
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")

	ctx := ctxinterrupt.WithCancelOnInterrupt(context.Background())
	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

func Main(version string) cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		cfg := config.NewRunCmdConfig(cliCtx)
		if err := cfg.Check(); err != nil {
			return fmt.Errorf("invalid CLI flags: %w", err)
		}

		l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.LogConfig())
		oplog.SetGlobalLogHandler(l.Handler())
		opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, l)

		s := runner.NewService(version, cfg, l)

		return s.Run(cliCtx.Context)
	}
}
