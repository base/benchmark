package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/base/base-bench/benchmark/config"
	"github.com/base/base-bench/benchmark/flags"
	"github.com/base/base-bench/runner"
	"github.com/base/base-bench/runner/importer"
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
		{
			Name:        "import-runs",
			Flags:       cliapp.ProtectFlags(flags.ImportRunsFlags),
			Action:      ImportMain(Version),
			Usage:       "import runs from metadata file or URL",
			Description: "Import benchmark runs from local metadata.json or remote URL into existing output metadata.json. Use --src-tag and --dest-tag to apply tags to runs, or use interactive mode.",
			ArgsUsage:   "[metadata-file-or-url]",
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

func ImportMain(version string) cli.ActionFunc {
	return func(cliCtx *cli.Context) error {
		cfg := config.NewImportCmdConfig(cliCtx)
		if err := cfg.Check(); err != nil {
			return fmt.Errorf("invalid CLI flags: %w", err)
		}

		l := oplog.NewLogger(oplog.AppOut(cliCtx), oplog.DefaultCLIConfig())
		oplog.SetGlobalLogHandler(l.Handler())

		service := importer.NewService(cfg, l)

		// Load source metadata
		srcMetadata, err := service.LoadSourceMetadata(cfg.SourceFile())
		if err != nil {
			return fmt.Errorf("failed to load source metadata: %w", err)
		}

		// Load destination metadata
		destMetadata, err := service.LoadDestinationMetadata()
		if err != nil {
			return fmt.Errorf("failed to load destination metadata: %w", err)
		}

		// Get preliminary summary for interactive mode
		prelimSummary := &importer.ImportSummary{
			ImportedRunsCount: len(srcMetadata.Runs),
			ExistingRunsCount: len(destMetadata.Runs),
		}

		// Check for conflicts
		existingIDs := make(map[string]bool)
		for _, run := range destMetadata.Runs {
			existingIDs[run.ID] = true
		}
		for _, run := range srcMetadata.Runs {
			if existingIDs[run.ID] {
				prelimSummary.Conflicts = append(prelimSummary.Conflicts, run.ID)
			}
		}

		// Determine tags and BenchmarkRun strategy - either from flags or interactive prompt
		srcTag := cfg.SrcTag()
		destTag := cfg.DestTag()
		benchmarkRunOpt := importer.BenchmarkRunCreateNew // Default strategy

		// Check if we need interactive mode
		needsInteractive := !cfg.NoConfirm() && (srcTag == nil || destTag == nil)

		if needsInteractive {
			// Run interactive prompt - this will determine strategy and tags if needed
			l.Info("Running interactive mode for import configuration")
			interactiveSrcTag, interactiveDestTag, interactiveBenchmarkRunOpt, confirmed, err := importer.RunInteractive(prelimSummary, destMetadata)
			if err != nil {
				return fmt.Errorf("interactive mode failed: %w", err)
			}
			if !confirmed {
				return fmt.Errorf("import cancelled")
			}

			// Use interactive values
			srcTag = interactiveSrcTag
			destTag = interactiveDestTag
			benchmarkRunOpt = interactiveBenchmarkRunOpt
		} else if len(destMetadata.Runs) > 0 {
			// Non-interactive mode with existing runs - default to adding to last run
			benchmarkRunOpt = importer.BenchmarkRunAddToLast
		}

		// Perform the import
		request := &importer.ImportRequest{
			SourceMetadata:  srcMetadata,
			DestMetadata:    destMetadata,
			SrcTag:          srcTag,
			DestTag:         destTag,
			BenchmarkRunOpt: benchmarkRunOpt,
			NoConfirm:       cfg.NoConfirm(),
		}

		result, err := service.Import(request)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}

		// Display results
		fmt.Printf("✅ Import completed successfully!\n")
		fmt.Printf("   • Imported: %d runs\n", result.ImportedRuns)
		fmt.Printf("   • Total runs: %d\n", result.TotalRuns)

		// Show if we downloaded files from URL
		if strings.HasPrefix(cfg.SourceFile(), "http://") || strings.HasPrefix(cfg.SourceFile(), "https://") {
			fmt.Printf("   • Downloaded output files from URL\n")
		}

		// Display BenchmarkRun strategy
		if benchmarkRunOpt == importer.BenchmarkRunAddToLast {
			fmt.Printf("   • Strategy: Added to existing run\n")
			if srcTag != nil {
				fmt.Printf("   • Applied source tag: %s=%s\n", srcTag.Key, srcTag.Value)
			}
			if destTag != nil {
				fmt.Printf("   • Applied destination tag: %s=%s\n", destTag.Key, destTag.Value)
			}
		} else {
			fmt.Printf("   • Strategy: Created new run group\n")
			fmt.Printf("   • Imported runs differentiated by BenchmarkRun ID\n")
		}

		return nil
	}
}
