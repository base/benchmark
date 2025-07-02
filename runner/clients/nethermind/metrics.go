package nethermind

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

func (n *metricsCollector) GetMetricsEndpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d/metrics", n.metricsPort)
}

func (n *metricsCollector) GetMetrics() []metrics.BlockMetrics {
	return n.metrics
}

func (n *metricsCollector) GetMetricTypes() map[string]bool {
	return map[string]bool{
		"nethermind_block_processing_time_seconds":       true,
		"nethermind_transaction_processing_time_seconds": true,
		"nethermind_state_db_reads_total":                true,
		"nethermind_state_db_writes_total":               true,
		"nethermind_memory_usage_bytes":                  true,
		"nethermind_peers_count":                         true,
		"nethermind_sync_peers_count":                    true,
		"nethermind_chain_head_number":                   true,
	}
}

func (n *metricsCollector) Collect(ctx context.Context, m *metrics.BlockMetrics) error {
	resp, err := http.Get(n.GetMetricsEndpoint())
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
	metricsData, err := txtParser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to parse metrics: %w", err)
	}

	metricTypes := n.GetMetricTypes()

	for _, metric := range metricsData {
		name := metric.GetName()
		if metricTypes[name] {
			metricVal := metric.GetMetric()
			if len(metricVal) != 1 {
				n.log.Warn("expected 1 metric, got %d for metric %s", len(metricVal), name)
			}
			err = m.AddPrometheusMetric(name, metricVal[0])
			if err != nil {
				n.log.Warn("failed to add metric %s: %s", name, err)
			}
		}
	}

	n.metrics = append(n.metrics, *m)
	return nil
}
