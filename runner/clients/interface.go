package clients

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/base/base-bench/runner/clients/geth"
	"github.com/base/base-bench/runner/clients/reth"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
)

// NewClient creates a new client based on the given client type.
func NewClient(client Client, logger log.Logger, options *config.ClientOptions) types.ExecutionClient {
	switch client {
	case Reth:
		return reth.NewRethClient(logger, options)
	case Geth:
		return geth.NewGethClient(logger, options)
	default:
		panic("unknown client")
	}
}

// Client is an enum for the different clients that can be used to run the chain.
type Client uint

const (
	Reth Client = iota
	Geth
)
