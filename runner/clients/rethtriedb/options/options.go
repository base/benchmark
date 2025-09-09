package options

import rethoptions "github.com/base/base-bench/runner/clients/reth/options"

// RethTriedbOptions contains the options for the reth-triedb client determined by the test.
type RethTriedbOptions struct {
	rethoptions.RethOptions

	// RethTriedbBin is the path to the reth-triedb binary.
	RethTriedbBin string
}
