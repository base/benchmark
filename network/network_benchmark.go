package network

import (
	"context"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Metrics interface{}
type Logs interface{}

type NetworkBenchmark struct {
	client    *ethclient.Client
	params    benchmark.Params
	jwtSecret string

	cl *FakeConsensusClient
}

func NewNetworkBenchmark(params benchmark.Params, client *ethclient.Client, genesis core.Genesis, jwtSecret string) *NetworkBenchmark {
	genesisHash := genesis.ToBlock().Hash()
	return &NetworkBenchmark{
		client: client,
		params: params,
		cl:     NewFakeConsensusClient(client, genesisHash, jwtSecret),
	}
}

func (nb *NetworkBenchmark) Run(ctx context.Context) error {
	err := nb.cl.Start(ctx)
	if err != nil {
		return err
	}

	err = nb.warmNetwork()
	if err != nil {
		return err
	}

	for range nb.params.TransactionPayload {
		err = nb.startPayloadWorker()
		if err != nil {
			return err
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
