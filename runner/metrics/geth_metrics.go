package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type GethMetricsCollector struct {
	log     log.Logger
	client  *ethclient.Client
	metrics []Metrics
}

func NewGethMetricsCollector(log log.Logger, client *ethclient.Client) *GethMetricsCollector {
	return &GethMetricsCollector{
		log:     log,
		client:  client,
		metrics: make([]Metrics, 0),
	}
}

func (g *GethMetricsCollector) GetMetricTypes() map[string]bool {
	return map[string]bool{
		"chain/account/commits.count":         true,
		"chain/account/commits.50-percentile": true,
		"chain/account/commits.95-percentile": true,
		"eth/db/chaindata/disk/read":          true,
		"eth/db/chaindata/disk/write":         true,
	}
}

func (g *GethMetricsCollector) GetMetricsEndpoint() string {
	return "http://127.0.0.1:8080/debug/metrics"
}

func (g *GethMetricsCollector) GetMetrics() []Metrics {
	return g.metrics
}

func (g *GethMetricsCollector) Collect(ctx context.Context) error {
	resp, err := http.Get(g.GetMetricsEndpoint())
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var metricsData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metricsData); err != nil {
		return fmt.Errorf("failed to decode metrics: %w", err)
	}

	block, err := g.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get block number: %w", err)
	}

	m := NewMetrics()
	m.BlockNumber = block

	metricTypes := g.GetMetricTypes()
	for name, value := range metricsData {
		if !metricTypes[name] {
			continue
		}
		if v, ok := value.(float64); ok {
			m.AddExecutionMetric(name, v)
		}
	}

	g.metrics = append(g.metrics, *m)
	return nil
}
