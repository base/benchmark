package rbuilder

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/log"

	"github.com/base/base-bench/runner/clients/reth"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/ethereum/go-ethereum/ethclient"
)

// RbuilderClient handles the lifecycle of a reth client.
type RbuilderClient struct {
	logger  log.Logger
	options *config.InternalClientOptions

	// client          *ethclient.Client
	// clientURL       string
	// authClient      client.RPC
	// rbuilderProcess *exec.Cmd

	// stdout io.WriteCloser
	// stderr io.WriteCloser

	elClient types.ExecutionClient
}

// NewRbuilderClient creates a new client for reth.
func NewRbuilderClient(logger log.Logger, options *config.InternalClientOptions) types.ExecutionClient {
	// only support reth for now
	rethClient := reth.NewRethClient(logger, options)

	return &RbuilderClient{
		logger:   logger,
		options:  options,
		elClient: rethClient,
	}
}

// Run runs the reth client with the given runtime config.
func (r *RbuilderClient) Run(ctx context.Context, cfg *types.RuntimeConfig) error {
	return r.elClient.Run(ctx, cfg)
}

// Stop stops the reth client.
func (r *RbuilderClient) Stop() {
	r.elClient.Stop()
}

// Client returns the ethclient client.
func (r *RbuilderClient) Client() *ethclient.Client {
	return r.elClient.Client()
}

// ClientURL returns the raw client URL for transaction generators.
func (r *RbuilderClient) ClientURL() string {
	return r.elClient.ClientURL()
}

// AuthClient returns the auth client used for CL communication.
func (r *RbuilderClient) AuthClient() client.RPC {
	return r.elClient.AuthClient()
}

func (r *RbuilderClient) MetricsPort() int {
	return r.elClient.MetricsPort()
}
