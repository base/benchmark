package rbuilderoptions

import "github.com/base/base-bench/runner/clients/reth/options"

// RethOptions contains the options for the reth client determined by the test.
type RbuilderOptions struct {
	options.RethOptions

	// RbuilderBin is the path to the rbuilder binary.
	RbuilderBin string
}
