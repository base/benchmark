package builderoptions

import "github.com/base/base-bench/runner/clients/reth/options"

// RethOptions contains the options for the reth client determined by the test.
type BuilderOptions struct {
	options.RethOptions

	// BuilderBin is the path to the builder binary.
	BuilderBin string
}
