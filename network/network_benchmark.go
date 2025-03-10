package network

import (
	"context"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Metrics interface{}
type Logs interface{}

type NetworkBenchmark struct {
	client     *ethclient.Client
	authClient client.RPC

	params    benchmark.Params
	jwtSecret [32]byte

	cl *FakeConsensusClient
}

func NewNetworkBenchmark(params benchmark.Params, client *ethclient.Client, authClient client.RPC, genesis core.Genesis) *NetworkBenchmark {
	genesisHash := genesis.ToBlock().Hash()
	return &NetworkBenchmark{
		client:     client,
		authClient: authClient,
		params:     params,
		cl:         NewFakeConsensusClient(client, authClient, genesisHash, genesis.Timestamp),
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
