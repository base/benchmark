package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/common/expfmt"
)

type RethMetricsCollector struct {
	log     log.Logger
	client  *ethclient.Client
	metrics []Metrics
}

func NewRethMetricsCollector(log log.Logger, client *ethclient.Client) *RethMetricsCollector {
	return &RethMetricsCollector{
		log:     log,
		client:  client,
		metrics: make([]Metrics, 0),
	}
}

func (r *RethMetricsCollector) GetMetricsEndpoint() string {
	return "http://localhost:8080/metrics"
}

func (r *RethMetricsCollector) GetMetrics() []Metrics {
	return r.metrics
}

func (r *RethMetricsCollector) GetMetricTypes() map[string]bool {
	return map[string]bool{
		"reth_sync_execution_execution_duration":         true,
		"reth_sync_block_validation_state_root_duration": true,
	}
}

func (r *RethMetricsCollector) Collect(ctx context.Context) error {
	resp, err := http.Get(r.GetMetricsEndpoint())
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read metrics response: %w", err)
	}

	txtParser := expfmt.TextParser{}
	metrics, err := txtParser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to parse metrics: %w", err)
	}

	block, err := r.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get block number: %w", err)
	}

	r.log.Info("Block number", "number", block)

	m := NewMetrics()
	m.BlockNumber = block

	metricTypes := r.GetMetricTypes()

	for _, metric := range metrics {
		name := metric.GetName()
		if metricTypes[name] {
			m.AddExecutionMetric(name, metric.GetMetric())
		}
	}

	r.metrics = append(r.metrics, *m)
	return nil
}
