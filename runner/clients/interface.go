package clients

import (
	"fmt"

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
type Client string

const (
	Reth Client = "reth"
	Geth Client = "geth"
)

var (
	ErrInvalidClient = fmt.Errorf("invalid client type")
)

func ParseClient(client string) (Client, error) {
	switch client {
	case "reth":
		return Reth, nil
	case "geth":
		return Geth, nil
	default:
		return "", ErrInvalidClient
	}
}
