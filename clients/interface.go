package clients

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/base/base-bench/clients/geth"
	"github.com/base/base-bench/clients/reth"
	"github.com/base/base-bench/clients/types"
)

func NewClient(client types.Client, logger log.Logger, options *types.ClientOptions) types.ExecutionClient {
	switch client {
	case types.Reth:
		return reth.NewRethClient(logger, options)
	case types.Geth:
		return geth.NewGethClient(logger, options)
	default:
		panic("unknown client")
	}
}
