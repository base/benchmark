package network

import (
	"fmt"

	"github.com/base/base-bench/runner/clients/geth"
	"github.com/base/base-bench/runner/clients/rbuilder"
	"github.com/base/base-bench/runner/clients/reth"
	"github.com/base/base-bench/runner/metrics"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

func NewMetricsCollector(
	log log.Logger,
	client *ethclient.Client,
	clientName string,
	metricsPort int) metrics.Collector {
	switch clientName {
	case "geth":
		return geth.NewMetricsCollector(log, client, metricsPort)
	case "reth":
		return reth.NewMetricsCollector(log, client, metricsPort)
	case "rbuilder":
		return rbuilder.NewMetricsCollector(log, client, metricsPort)
	}
	panic(fmt.Sprintf("unknown client: %s", clientName))
}
