package benchmark

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/base/base-bench/runner/payload"
)

type BenchmarkRole string

const (
	// BenchmarkRoleSequencer is always required. Every benchmark starts by
	// running the sequencer phase, which builds the payloads consumed by any
	// later validator phase.
	BenchmarkRoleSequencer BenchmarkRole = "sequencer"

	// BenchmarkRoleValidator is optional. When enabled, the validator phase
	// replays the payloads produced by the sequencer phase.
	BenchmarkRoleValidator BenchmarkRole = "validator"
)

// BenchmarkExecutionMode is the normalized internal execution model.
//
// The YAML config exposes "roles", but the runner does not support arbitrary
// role combinations: the sequencer phase always runs, and the only real choice
// is whether to also run the validator phase after it.
type BenchmarkExecutionMode struct {
	RunValidator bool
}

var defaultBenchmarkExecutionMode = BenchmarkExecutionMode{RunValidator: true}

func BenchmarkExecutionModeFromRoles(roles []BenchmarkRole) (BenchmarkExecutionMode, error) {
	if len(roles) == 0 {
		return defaultBenchmarkExecutionMode, nil
	}

	seen := make(map[BenchmarkRole]bool, len(roles))
	for _, role := range roles {
		switch role {
		case BenchmarkRoleSequencer, BenchmarkRoleValidator:
		default:
			return BenchmarkExecutionMode{}, fmt.Errorf("invalid benchmark role %q", role)
		}

		if seen[role] {
			return BenchmarkExecutionMode{}, fmt.Errorf("duplicate benchmark role %q", role)
		}
		seen[role] = true
	}

	if !seen[BenchmarkRoleSequencer] {
		return BenchmarkExecutionMode{}, fmt.Errorf("benchmark roles must include %q", BenchmarkRoleSequencer)
	}

	// A validator-only benchmark is invalid because the validator phase consumes
	// payloads and setup state produced by the sequencer phase.
	return BenchmarkExecutionMode{RunValidator: seen[BenchmarkRoleValidator]}, nil
}

// Roles returns the config-facing role list for metadata and logs. Internally,
// callers should use RunValidator instead of reinterpreting the role slice.
func (mode BenchmarkExecutionMode) Roles() []BenchmarkRole {
	roles := []BenchmarkRole{BenchmarkRoleSequencer}
	if mode.RunValidator {
		roles = append(roles, BenchmarkRoleValidator)
	}
	return roles
}

func (mode BenchmarkExecutionMode) RolesString() string {
	names := make([]string, 0, 2)
	for _, role := range mode.Roles() {
		names = append(names, string(role))
	}
	return strings.Join(names, ",")
}

func (mode BenchmarkExecutionMode) IsDefault() bool {
	return mode == defaultBenchmarkExecutionMode
}

// Param is a single dimension of a benchmark matrix. It can be a
// single value or a list of values.
type Param struct {
	Name      *string       `yaml:"name"`
	ParamType string        `yaml:"type"`
	Value     interface{}   `yaml:"value"`
	Values    []interface{} `yaml:"values"`
}

func (bp *Param) Check() error {
	if bp.Value == nil && bp.Values == nil {
		return errors.New("value or values is required")
	}
	if bp.Value != nil && bp.Values != nil {
		return errors.New("value and values cannot both be specified")
	}
	return nil
}

type ProofProgramOptions struct {
	Enabled *bool  `yaml:"enabled"`
	Version string `yaml:"version"`
	Type    string `yaml:"type"`
}

// SnapshotDefinition is the user-facing YAML configuration for specifying
// a snapshot to be restored before running a benchmark.
type SnapshotDefinition struct {
	Command           string  `yaml:"command"`
	GenesisFile       *string `yaml:"genesis_file"`
	SuperchainChainID *uint64 `yaml:"superchain_chain_id"`
	ForceClean        *bool   `yaml:"force_clean"`
}

// CreateSnapshot copies the snapshot to the output directory for the given
// node type.
func (s SnapshotDefinition) CreateSnapshot(nodeType string, outputDir string) error {
	// default to true if not set
	forceClean := s.ForceClean == nil || *s.ForceClean
	if _, err := os.Stat(outputDir); err == nil && forceClean {
		// TODO: we could reuse it here potentially
		if err := os.RemoveAll(outputDir); err != nil {
			return fmt.Errorf("failed to remove existing snapshot: %w", err)
		}
	}

	// get absolute path of outputDir
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get absolute path of outputDir: %w", err)
	}

	if !filepath.IsAbs(outputDir) {
		outputDir = path.Join(currentDir, outputDir)
	}

	var cmdBin string
	var args []string
	// split out default args from command
	parts := strings.SplitN(s.Command, " ", 2)
	if len(parts) < 2 {
		cmdBin = parts[0]
	} else {
		cmdBin = parts[0]
		args = strings.Split(parts[1], " ")
	}

	args = append(args, nodeType, outputDir)

	cmd := exec.Command(cmdBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// FlashblocksConfig holds top-level flashblocks configuration.
type FlashblocksConfig struct {
	BlockTime  string `yaml:"block_time"`
	LeewayTime string `yaml:"leeway_time"`
}

const DefaultFlashblocksBlockTime = "250"
const DefaultBlockTime = "1s"

type BenchmarkConfig struct {
	Name                string               `yaml:"name"`
	Description         *string              `yaml:"description"`
	BlockTime           *string              `yaml:"block_time"`
	Flashblocks         *FlashblocksConfig   `yaml:"flashblocks"`
	Benchmarks          []TestDefinition     `yaml:"benchmarks"`
	TransactionPayloads []payload.Definition `yaml:"payloads"`
}

// GetBlockTime returns the configured block time as a duration, or the default (1s).
func (bc *BenchmarkConfig) GetBlockTime() (time.Duration, error) {
	raw := DefaultBlockTime
	if bc.BlockTime != nil && *bc.BlockTime != "" {
		raw = *bc.BlockTime
	}
	return time.ParseDuration(raw)
}

// FlashblocksBlockTime returns the configured flashblocks block time, or the default.
func (bc *BenchmarkConfig) FlashblocksBlockTime() string {
	if bc.Flashblocks != nil && bc.Flashblocks.BlockTime != "" {
		return bc.Flashblocks.BlockTime
	}
	return DefaultFlashblocksBlockTime
}

// FlashblocksLeewayTime returns the configured flashblocks leeway time, if set.
func (bc *BenchmarkConfig) FlashblocksLeewayTime() string {
	if bc.Flashblocks != nil && bc.Flashblocks.LeewayTime != "" {
		return bc.Flashblocks.LeewayTime
	}
	return ""
}

type DatadirConfig struct {
	Sequencer *string `yaml:"sequencer"`
	Validator *string `yaml:"validator"`
}

// TestDefinition is the user-facing YAML configuration for specifying a
// matrix of benchmark runs.
type TestDefinition struct {
	Datadir      *DatadirConfig       `yaml:"datadirs"`
	Snapshot     *SnapshotDefinition  `yaml:"snapshot"`
	Metrics      *ThresholdConfig     `yaml:"metrics"`
	Tags         *map[string]string   `yaml:"tags"`
	Roles        []BenchmarkRole      `yaml:"roles"`
	Variables    []Param              `yaml:"variables"`
	ProofProgram *ProofProgramOptions `yaml:"proof_program"`
}

func (bc *TestDefinition) Check() error {
	mode, err := bc.ExecutionMode()
	if err != nil {
		return err
	}

	proofProgramEnabled := bc.ProofProgram != nil && (bc.ProofProgram.Enabled == nil || *bc.ProofProgram.Enabled)
	if proofProgramEnabled && !mode.RunValidator {
		return errors.New("proof_program requires the validator benchmark role")
	}

	if err := bc.validateThresholdRoles(mode); err != nil {
		return err
	}

	for _, b := range bc.Variables {
		err := b.Check()
		if err != nil {
			return err
		}
	}
	return nil
}

func (bc *TestDefinition) ExecutionMode() (BenchmarkExecutionMode, error) {
	return BenchmarkExecutionModeFromRoles(bc.Roles)
}

func (bc *TestDefinition) validateThresholdRoles(mode BenchmarkExecutionMode) error {
	if bc.Metrics == nil {
		return nil
	}

	for level, thresholds := range map[string]map[string]float64{
		"warning": bc.Metrics.Warning,
		"error":   bc.Metrics.Error,
	} {
		for metric := range thresholds {
			role, _, ok := strings.Cut(metric, "/")
			if !ok {
				continue
			}

			if BenchmarkRole(role) == BenchmarkRoleValidator && !mode.RunValidator {
				return fmt.Errorf("%s threshold %q requires the validator benchmark role", level, metric)
			}
		}
	}

	return nil
}
