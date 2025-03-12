package network

import (
	"context"
	"math/big"
	"sync"

	"github.com/base/base-bench/payload"
	"github.com/base/base-bench/runner/benchmark"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type Metrics interface{}
type Logs interface{}

type NetworkBenchmark struct {
	log log.Logger

	client     *ethclient.Client
	authClient client.RPC
	worker     payload.Worker

	params    benchmark.Params
	jwtSecret [32]byte

	cl *FakeConsensusClient
}

func NewNetworkBenchmark(log log.Logger, benchParams benchmark.Params, client *ethclient.Client, clientRPCURL string, authClient client.RPC, genesis core.Genesis) (*NetworkBenchmark, error) {
	genesisHash := genesis.ToBlock().Hash()
	amount := new(big.Int).Mul(big.NewInt(1e6), big.NewInt(params.Ether))

	worker, err := payload.NewTransferPayloadWorker(log, clientRPCURL, benchParams, common.FromHex("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"), amount)
	if err != nil {
		return nil, err
	}

	return &NetworkBenchmark{
		log:        log,
		client:     client,
		authClient: authClient,
		// TODO: pass this in somehow
		worker: worker,
		params: benchParams,
		cl:     NewFakeConsensusClient(log, client, authClient, genesisHash, genesis.Timestamp),
	}, nil
}

func (nb *NetworkBenchmark) Run(ctx context.Context) error {
	wg := sync.WaitGroup{}
	wg.Add(2)
	errChan := make(chan error)

	go func() {
		err := nb.cl.Start(ctx)
		if err != nil {
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

		err = nb.worker.Start(ctx)
		if err != nil {
			nb.log.Warn("failed to start payload worker", "err", err)
		}
		errChan <- err
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (nb *NetworkBenchmark) warmNetwork() error {
	return nil
}

func (nb *NetworkBenchmark) startPayloadWorker() error {
	return nil
}

func (nb *NetworkBenchmark) CollectResults() (Metrics, Logs) {
	return nil, nil
}
