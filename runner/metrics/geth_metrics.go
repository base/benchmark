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
	log         log.Logger
	client      *ethclient.Client
	metrics     []BlockMetrics
	metricsPort int
}

func NewGethMetricsCollector(log log.Logger, client *ethclient.Client, metricsPort int) *GethMetricsCollector {
	return &GethMetricsCollector{
		log:         log,
		client:      client,
		metricsPort: metricsPort,
		metrics:     make([]BlockMetrics, 0),
	}
}

func (g *GethMetricsCollector) GetMetricTypes() map[string]bool {
	return map[string]bool{
		"chain/account/reads.50-percentile":    true,
		"chain/execution.50-percentile":        true,
		"chain/crossvalidation.50-percentile":  true,
		"chain/storage/reads.50-percentile":    true,
		"chain/account/updates.50-percentile":  true,
		"chain/account/hashes.50-percentile":   true,
		"chain/storage/updates.50-percentile":  true,
		"chain/validation.50-percentile":       true,
		"chain/write.50-percentile":            true,
		"chain/snapshot/commits.50-percentile": true,
		"chain/triedb/commits.50-percentile":   true,
		"chain/account/commits.50-percentile":  true,
		"chain/storage/commits.50-percentile":  true,
		"chain/inserts.50-percentile":          true,
	}
}

func (g *GethMetricsCollector) GetMetricsEndpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d/debug/metrics", g.metricsPort)
}

func (g *GethMetricsCollector) GetMetrics() []BlockMetrics {
	return g.metrics
}

func (g *GethMetricsCollector) Collect(ctx context.Context, metrics *BlockMetrics) error {
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

	metricTypes := g.GetMetricTypes()
	for name, value := range metricsData {
		if !metricTypes[name] {
			continue
		}
		if v, ok := value.(float64); ok {
			metrics.AddExecutionMetric(name, v)
		}
	}

	g.metrics = append(g.metrics, *metrics)
	return nil
}
