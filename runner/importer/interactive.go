package importer

import (
	"fmt"
	"strings"

	"github.com/base/base-bench/benchmark/config"
	"github.com/base/base-bench/runner/benchmark"
	"github.com/charmbracelet/huh"
)

// TagFormData holds the form data for tag configuration
type TagFormData struct {
	BenchmarkRunMode string // "last" or "new"
	SrcKey           string
	SrcValue         string
	DestValue        string // Same key as source, only value differs
	NeedsDestTag     bool   // Whether destination tag is needed
	Confirm          bool
}

// checkSourceTagCoverage checks if the source tag key exists on all runs in the metadata
func checkSourceTagCoverage(destMetadata *benchmark.RunGroup, tagKey string) (bool, int, int) {
	if len(destMetadata.Runs) == 0 {
		return true, 0, 0 // No runs to check
	}

	runsWithTag := 0
	totalRuns := len(destMetadata.Runs)

	for _, run := range destMetadata.Runs {
		if run.TestConfig != nil {
			if _, exists := run.TestConfig[tagKey]; exists {
				runsWithTag++
			}
		}
	}

	allHaveTag := runsWithTag == totalRuns
	return allHaveTag, runsWithTag, totalRuns
}

// RunInteractive runs the interactive prompt and returns the configured tags
func RunInteractive(summary *ImportSummary, destMetadata *benchmark.RunGroup) (*config.TagConfig, *config.TagConfig, BenchmarkRunOption, bool, error) {
	var formData TagFormData

	// Show import summary first
	fmt.Printf("🚀 Import Benchmark Runs\n\n")
	fmt.Printf("📊 Import Summary:\n")
	fmt.Printf("  • Existing runs: %d\n", summary.ExistingRunsCount)
	fmt.Printf("  • Importing runs: %d\n", summary.ImportedRunsCount)

	if len(summary.Conflicts) > 0 {
		fmt.Printf("  ⚠️  Conflicts detected: %d run IDs already exist\n", len(summary.Conflicts))
	}
	fmt.Printf("\n")

	// First, ask about BenchmarkRun strategy
	strategyForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Import Strategy").
				Description("How should the imported runs be grouped?").
				Options(
					huh.NewOption("Add to last existing run (requires tags for differentiation)", "last").
						Selected(summary.ExistingRunsCount > 0), // Default to "last" if there are existing runs
					huh.NewOption("Create new separate run (differentiated by BenchmarkRun ID)", "new").
						Selected(summary.ExistingRunsCount == 0), // Default to "new" if no existing runs
				).
				Value(&formData.BenchmarkRunMode),
		).Title("🔄 Import Strategy Selection"),
	)

	if err := strategyForm.Run(); err != nil {
		return nil, nil, BenchmarkRunCreateNew, false, fmt.Errorf("strategy selection cancelled: %w", err)
	}

	var forms []*huh.Form

	// Only ask for tags if adding to last run
	if formData.BenchmarkRunMode == "last" {
		// First, ask for source tag
		sourceTagForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Tag Key").
					Description("Tag key to differentiate runs (will be applied to both existing and imported runs)").
					Placeholder("instance").
					Value(&formData.SrcKey).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("tag key is required when adding to last run")
						}
						return nil
					}),

				huh.NewInput().
					Title("Tag Value for Existing Runs").
					Description("Tag value to apply to existing runs (will only fill missing tags)").
					Placeholder("i7ie.24xlarge").
					Value(&formData.SrcValue).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("tag value for existing runs is required")
						}
						return nil
					}),
			).Title("📋 Configure tag for existing runs"),
		)
		// Run the source tag form to get the key, then check coverage
		if err := sourceTagForm.Run(); err != nil {
			return nil, nil, BenchmarkRunCreateNew, false, fmt.Errorf("source tag form cancelled: %w", err)
		}

		// Check if we need destination tag
		allHaveTag, runsWithTag, totalRuns := checkSourceTagCoverage(destMetadata, strings.TrimSpace(formData.SrcKey))
		formData.NeedsDestTag = !allHaveTag

		if formData.NeedsDestTag {
			fmt.Printf("ℹ️  Tag coverage analysis:\n")
			fmt.Printf("   • Runs with '%s' tag: %d/%d\n", formData.SrcKey, runsWithTag, totalRuns)
			fmt.Printf("   • Need destination tag to differentiate imported runs\n\n")

			destTagForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title(fmt.Sprintf("Tag Value for Imported Runs (key: %s)", formData.SrcKey)).
						Description("Tag value to apply to imported runs").
						Placeholder("i7ie.8xlarge").
						Value(&formData.DestValue).
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("tag value for imported runs is required")
							}
							return nil
						}),
				).Title("📋 Configure tag for imported runs"),
			)
			forms = append(forms, destTagForm)
		} else {
			fmt.Printf("ℹ️  Tag coverage analysis:\n")
			fmt.Printf("   • All existing runs have '%s' tag\n", formData.SrcKey)
			fmt.Printf("   • No destination tag needed for imported runs\n\n")
		}
	} else {
		// For new runs, show informational message
		fmt.Printf("ℹ️  Creating new run group - no additional tags needed\n")
		fmt.Printf("   Imported runs will be differentiated by their BenchmarkRun ID\n\n")
	}

	// Always add confirmation
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Confirm Import").
				Description(func() string {
					if formData.BenchmarkRunMode == "last" {
						if formData.NeedsDestTag {
							return "Do you want to proceed with adding to the last run using the configured tags?"
						}
						return "Do you want to proceed with adding to the last run (filling missing source tags only)?"
					}
					return "Do you want to proceed with creating a new run group?"
				}()).
				Value(&formData.Confirm),
		).Title("✅ Confirmation"),
	)
	forms = append(forms, confirmForm)

	// Run the remaining forms
	for _, form := range forms {
		fmt.Printf("Running form: %v\n", form)
		if err := form.Run(); err != nil {
			return nil, nil, BenchmarkRunCreateNew, false, fmt.Errorf("form cancelled: %w", err)
		}
	}

	// If user didn't confirm, return cancelled
	if !formData.Confirm {
		return nil, nil, BenchmarkRunCreateNew, false, fmt.Errorf("import cancelled by user")
	}

	// Build tag configs (only if adding to last run)
	var srcTag, destTag *config.TagConfig

	if formData.BenchmarkRunMode == "last" {
		if strings.TrimSpace(formData.SrcKey) != "" {
			srcTag = &config.TagConfig{
				Key:   strings.TrimSpace(formData.SrcKey),
				Value: strings.TrimSpace(formData.SrcValue),
			}

			// Only create destination tag if needed
			if formData.NeedsDestTag && strings.TrimSpace(formData.DestValue) != "" {
				destTag = &config.TagConfig{
					Key:   strings.TrimSpace(formData.SrcKey),
					Value: strings.TrimSpace(formData.DestValue),
				}
			}
		}
	}

	// Determine BenchmarkRun option
	var benchmarkRunOpt BenchmarkRunOption
	if formData.BenchmarkRunMode == "last" {
		benchmarkRunOpt = BenchmarkRunAddToLast
	} else {
		benchmarkRunOpt = BenchmarkRunCreateNew
	}

	return srcTag, destTag, benchmarkRunOpt, true, nil
}
