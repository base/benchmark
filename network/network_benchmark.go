package network

import (
	"context"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Metrics interface{}
type Logs interface{}

type NetworkBenchmark struct {
	client *ethclient.Client
	params benchmark.Params
}

func NewNetworkBenchmark(params benchmark.Params, client *ethclient.Client) *NetworkBenchmark {
	return &NetworkBenchmark{
		client,
		params,
	}
}

func (nb *NetworkBenchmark) Run(ctx context.Context) error {
	err := nb.warmNetwork()
	if err != nil {
		return err
	}

	for _, payload := range nb.params.TransactionPayload {
		err = nb.startPayloadWorker()
		if err != nil {
			return err
		}
	}


}

func (nb *NetworkBenchmark) warmNetwork() error {

}

func (nb *NetworkBenchmark) startPayloadWorker() error {

}

func (nb *NetworkBenchmark)

func (nb *NetworkBenchmark) CollectResults() (Metrics, Logs) {
	return nil, nil
}
