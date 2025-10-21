package rbuilderoptions

import "github.com/base/base-bench/runner/clients/reth/options"

// RbuilderOptions contains the options for the rbuilder client.
// Supports two modes:
// 1. Simple mode: Just rbuilder standalone (for testing)
// 2. Dual-builder mode: Fallback builder + rbuilder + rollup-boost (production architecture)
type RbuilderOptions struct {
	options.RethOptions

	// RbuilderBin is the path to the rbuilder binary (primary flashblock builder).
	RbuilderBin string

	// FallbackClient specifies which client to use as fallback builder.
	// If empty, runs in simple mode (rbuilder only).
	// Valid values: "geth", "reth"
	FallbackClient string

	// RollupBoostBin is the path to the rollup-boost coordinator binary.
	// If empty, no rollup-boost coordination is used.
	RollupBoostBin string
}
