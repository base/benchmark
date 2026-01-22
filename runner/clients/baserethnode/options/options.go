package baserethnode

import "github.com/base/base-bench/runner/clients/reth/options"

// BaseRethNodeOptions contains the options for the base-reth-node client determined by the test.
type BaseRethNodeOptions struct {
	options.RethOptions

	// BaseRethNodeBin is the path to the base-reth-node binary.
	BaseRethNodeBin string
}
