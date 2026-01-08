package rbuilder

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/log"

	"github.com/base/base-bench/runner/benchmark/portmanager"
	"github.com/base/base-bench/runner/clients/reth"
	"github.com/base/base-bench/runner/clients/types"
	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/metrics"
	"github.com/ethereum/go-ethereum/ethclient"
)

// RbuilderClient handles the lifecycle of a reth client.
type RbuilderClient struct {
	logger  log.Logger
	options *config.InternalClientOptions

	ports         portmanager.PortManager
	websocketPort uint64

	elClient types.ExecutionClient

	metricsCollector metrics.Collector
}

// NewRbuilderClient creates a new client for reth.
func NewRbuilderClient(logger log.Logger, options *config.InternalClientOptions, ports portmanager.PortManager) types.ExecutionClient {
	// only support reth for now
	rethClient := reth.NewRethClientWithBin(logger, options, ports, options.RbuilderBin)

	return &RbuilderClient{
		logger:   logger,
		options:  options,
		elClient: rethClient,
		ports:    ports,
	}
}

// Run runs the reth client with the given runtime config.
func (r *RbuilderClient) Run(ctx context.Context, cfg *types.RuntimeConfig) error {
	r.websocketPort = r.ports.AcquirePort("rbuilder", portmanager.FlashblocksWebsocketPortPurpose)

	cfg2 := *cfg
	cfg2.Args = append(cfg2.Args, "--flashblocks.enabled")
	cfg2.Args = append(cfg2.Args, "--flashblocks.port", fmt.Sprintf("%d", r.websocketPort))
	cfg2.Args = append(cfg2.Args, "--flashblocks.fixed")
	err := r.elClient.Run(ctx, &cfg2)
	if err != nil {
		return err
	}

	r.metricsCollector = newMetricsCollector(r.logger, r.elClient.Client(), int(r.elClient.MetricsPort()))
	if r.metricsCollector == nil {
		return errors.New("failed to create metrics collector")
	}
	return nil
}

func (r *RbuilderClient) MetricsCollector() metrics.Collector {
	return r.metricsCollector
}

// Stop stops the reth client.
func (r *RbuilderClient) Stop() {
	r.ports.ReleasePort(r.websocketPort)
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

// GetVersion returns the version of the Rbuilder client
func (r *RbuilderClient) GetVersion(ctx context.Context) (string, error) {
	// Rbuilder is based on reth, so delegate to the underlying reth client
	return r.elClient.GetVersion(ctx)
}

// SetHead resets the blockchain to a specific block using debug.setHead
func (r *RbuilderClient) SetHead(ctx context.Context, blockNumber uint64) error {
	// Rbuilder is based on reth, so delegate to the underlying reth client
	return r.elClient.SetHead(ctx, blockNumber)
}
