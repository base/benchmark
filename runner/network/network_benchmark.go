package network

import (
	"context"
	"errors"
	"math/big"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/payload"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type Metrics interface{}
type Logs interface{}

// NetworkBenchmark handles the lifecycle for a single benchmark run.
type NetworkBenchmark struct {
	log log.Logger

	client     *ethclient.Client
	authClient client.RPC
	worker     payload.Worker

	params benchmark.Params

	cl *FakeConsensusClient
}

// NewNetworkBenchmark creates a new network benchmark and initializes the payload worker and consensus client.
func NewNetworkBenchmark(log log.Logger, benchParams benchmark.Params, client *ethclient.Client, clientRPCURL string, authClient client.RPC, genesis *core.Genesis, metricsCollector metrics.MetricsCollector) (*NetworkBenchmark, error) {
	amount := new(big.Int).Mul(big.NewInt(1e6), big.NewInt(params.Ether))

	mempool, worker, err := payload.NewTransferPayloadWorker(log, clientRPCURL, benchParams, common.FromHex("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"), amount)
	if err != nil {
		return nil, err
	}

	consensusClient := NewFakeConsensusClient(log, client, authClient, mempool, genesis, metricsCollector, FakeConsensusClientOptions{
		BlockTime: benchParams.BlockTime,
	})

	return &NetworkBenchmark{
		log:        log,
		client:     client,
		authClient: authClient,
		worker:     worker,
		params:     benchParams,
		cl:         consensusClient,
	}, nil
}

// Run starts the benchmark by starting the consensus client and the payload worker.
func (nb *NetworkBenchmark) Run(ctx context.Context) error {
	errChan := make(chan error)

	consensusClientCtx, cancel := context.WithCancel(ctx)

	go func() {
		err := nb.cl.Start(consensusClientCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			nb.log.Warn("failed to run consensus client", "err", err)
		}
		errChan <- err
	}()

	go func() {
		err := nb.worker.Setup(ctx)
		if err != nil {
			nb.log.Warn("failed to setup payload worker", "err", err)
			errChan <- err
			return
		}

		err = nb.worker.Run(ctx)
		if err != nil {
			nb.log.Warn("failed to start payload worker", "err", err)
		}
		errChan <- err

		// once this finishes, we should cancel the consensus client
		cancel()
	}()

	// wait for both to finish or one to fail
	for i := 0; i < 2; i++ {
		err := <-errChan
		if err != nil {
			return err
		}
	}
	return nil
}

// CollectResults collects the metrics and logs from the benchmark run.
func (nb *NetworkBenchmark) CollectResults() (Metrics, Logs) {
	return nil, nil
}
