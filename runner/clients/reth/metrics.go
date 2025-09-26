package reth

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/base/base-bench/runner/metrics"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/common/expfmt"
)

type metricsCollector struct {
	log         log.Logger
	client      *ethclient.Client
	metrics     []metrics.BlockMetrics
	metricsPort int
}

func newMetricsCollector(log log.Logger, client *ethclient.Client, metricsPort int) metrics.Collector {
	return &metricsCollector{
		log:         log,
		client:      client,
		metricsPort: metricsPort,
		metrics:     make([]metrics.BlockMetrics, 0),
	}
}

func (r *metricsCollector) GetMetricsEndpoint() string {
	return fmt.Sprintf("http://localhost:%d/metrics", r.metricsPort)
}

func (r *metricsCollector) GetMetrics() []metrics.BlockMetrics {
	return r.metrics
}

func (r *metricsCollector) GetMetricTypes() map[string]bool {
	return map[string]bool{
		"reth_sync_execution_execution_duration":               true,
		"reth_sync_block_validation_state_root_duration":       true,
		"reth_sync_state_provider_storage_fetch_latency":       true,
		"reth_sync_state_provider_account_fetch_latency":       true,
		"reth_sync_state_provider_code_fetch_latency":          true,
		"reth_sync_state_provider_total_storage_fetch_latency": true,
		"reth_sync_state_provider_total_account_fetch_latency": true,
		"reth_sync_state_provider_total_code_fetch_latency":    true,
	}
}

func (r *metricsCollector) Collect(ctx context.Context, m *metrics.BlockMetrics) error {
	resp, err := http.Get(r.GetMetricsEndpoint())
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read metrics response: %w", err)
	}

	txtParser := expfmt.TextParser{}
	metrics, err := txtParser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to parse metrics: %w", err)
	}

	metricTypes := r.GetMetricTypes()

	for _, metric := range metrics {
		name := metric.GetName()
		if metricTypes[name] {
			metricVal := metric.GetMetric()
			if len(metricVal) != 1 {
				r.log.Warn("expected 1 metric, got %d for metric %s", len(metricVal), name)
			}
			err = m.UpdatePrometheusMetric(name, metricVal[0])
			if err != nil {
				r.log.Warn("failed to add metric %s: %s", name, err)
			}
		}
	}

	r.metrics = append(r.metrics, *m.Copy())
	return nil
}
