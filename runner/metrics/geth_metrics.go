package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
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
		"chain_account_commits": true,
		"chain_head_block":      true,
		"eth_peer_count":        true,
		"eth_txpool_pending":    true,
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

	block, err := g.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get block number: %w", err)
	}

	m := NewMetrics()
	m.BlockNumber = block

	metricTypes := g.GetMetricTypes()

	for name, metricFamily := range metrics {
		if !metricTypes[name] {
			continue
		}

		for _, metric := range metricFamily.GetMetric() {
			switch metricFamily.GetType() {
			case dto.MetricType_COUNTER, dto.MetricType_GAUGE:
				if counter := metric.GetCounter(); counter != nil {
					m.AddExecutionMetric(name, counter.GetValue())
				}
				if gauge := metric.GetGauge(); gauge != nil {
					m.AddExecutionMetric(name, gauge.GetValue())
				}
			}
		}
	}

	g.metrics = append(g.metrics, *m)
	g.log.Info("Collected metrics", "block", block)

	return nil
}
