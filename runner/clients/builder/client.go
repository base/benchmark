package builder

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

// BuilderClient handles the lifecycle of a builder client.
type BuilderClient struct {
	logger  log.Logger
	options *config.InternalClientOptions

	ports         portmanager.PortManager
	websocketPort uint64

	elClient          types.ExecutionClient
	flashblocksClient types.FlashblocksClient

	metricsCollector metrics.Collector
}

// NewBuilderClient creates a new builder client.
func NewBuilderClient(logger log.Logger, options *config.InternalClientOptions, ports portmanager.PortManager) types.ExecutionClient {
	// only support reth for now
	rethClient := reth.NewRethClientWithBin(logger, options, ports, options.BuilderBin)

	return &BuilderClient{
		logger:   logger,
		options:  options,
		elClient: rethClient,
		ports:    ports,
	}
}

// Run runs the builder client with the given runtime config.
func (r *BuilderClient) Run(ctx context.Context, cfg *types.RuntimeConfig) error {
	r.websocketPort = r.ports.AcquirePort("builder", portmanager.FlashblocksWebsocketPortPurpose)

	cfg2 := *cfg
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

	// Create flashblocks client
	r.flashblocksClient = NewFlashblocksClient(r.logger, r.websocketPort)

	return nil
}

func (r *BuilderClient) MetricsCollector() metrics.Collector {
	return r.metricsCollector
}

// Stop stops the builder client.
func (r *BuilderClient) Stop() {
	// Stop flashblocks client if it exists
	if r.flashblocksClient != nil {
		if err := r.flashblocksClient.Stop(); err != nil {
			r.logger.Warn("Failed to stop flashblocks client", "err", err)
		}
	}

	r.ports.ReleasePort(r.websocketPort)
	r.elClient.Stop()
}

// Client returns the ethclient client.
func (r *BuilderClient) Client() *ethclient.Client {
	return r.elClient.Client()
}

// ClientURL returns the raw client URL for transaction generators.
func (r *BuilderClient) ClientURL() string {
	return r.elClient.ClientURL()
}

// AuthClient returns the auth client used for CL communication.
func (r *BuilderClient) AuthClient() client.RPC {
	return r.elClient.AuthClient()
}

func (r *BuilderClient) MetricsPort() int {
	return r.elClient.MetricsPort()
}

// GetVersion returns the version of the builder client
func (r *BuilderClient) GetVersion(ctx context.Context) (string, error) {
	// Builder is based on reth, so delegate to the underlying reth client
	return r.elClient.GetVersion(ctx)
}

// SetHead resets the blockchain to a specific block using debug.setHead
func (r *BuilderClient) SetHead(ctx context.Context, blockNumber uint64) error {
	// Builder is based on reth, so delegate to the underlying reth client
	return r.elClient.SetHead(ctx, blockNumber)
}

// FlashblocksClient returns the flashblocks websocket client for collecting flashblock payloads.
func (r *BuilderClient) FlashblocksClient() types.FlashblocksClient {
	return r.flashblocksClient
}

// SupportsFlashblocks returns false as builder doesn't support receiving flashblock payloads.
func (r *BuilderClient) SupportsFlashblocks() bool {
	return false
}
