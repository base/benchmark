package baserethnode

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
	"github.com/prometheus/common/model"
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
		"reth_sync_execution_execution_duration":                          true,
		"reth_sync_block_validation_state_root_duration":                  true,
		"reth_sync_state_provider_storage_fetch_latency":                  true,
		"reth_sync_state_provider_account_fetch_latency":                  true,
		"reth_sync_state_provider_code_fetch_latency":                     true,
		"reth_sync_state_provider_total_storage_fetch_latency":            true,
		"reth_sync_state_provider_total_account_fetch_latency":            true,
		"reth_sync_state_provider_total_code_fetch_latency":               true,
		"reth_reth_flashblocks_upstream_errors":                           true,
		"reth_reth_flashblocks_upstream_messages":                         true,
		"reth_reth_flashblocks_block_processing_duration":                 true,
		"reth_reth_flashblocks_sender_recovery_duration":                  true,
		"reth_reth_flashblocks_unexpected_block_order":                    true,
		"reth_reth_flashblocks_flashblocks_in_block":                      true,
		"reth_reth_flashblocks_block_processing_error":                    true,
		"reth_reth_flashblocks_pending_clear_catchup":                     true,
		"reth_reth_flashblocks_pending_clear_reorg":                       true,
		"reth_reth_flashblocks_pending_snapshot_fb_index":                 true,
		"reth_reth_flashblocks_pending_snapshot_height":                   true,
		"reth_reth_flashblocks_reconnect_attempts":                        true,
		"reth_reth_flashblocks_rpc_get_transaction_count":                 true,
		"reth_reth_flashblocks_rpc_get_transaction_receipt":               true,
		"reth_reth_flashblocks_rpc_get_transaction_by_hash":               true,
		"reth_reth_flashblocks_rpc_get_balance":                           true,
		"reth_reth_flashblocks_rpc_get_block_by_number":                   true,
		"reth_reth_flashblocks_rpc_call":                                  true,
		"reth_reth_flashblocks_rpc_estimate_gas":                          true,
		"reth_reth_flashblocks_rpc_simulate_v1":                           true,
		"reth_reth_flashblocks_rpc_get_logs":                              true,
		"reth_reth_flashblocks_rpc_get_block_transaction_count_by_number": true,
		"reth_reth_flashblocks_bundle_state_clone_duration":               true,
		"reth_reth_flashblocks_bundle_state_clone_size":                   true,
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

	txtParser := expfmt.NewTextParser(model.LegacyValidation)
	parsedMetrics, err := txtParser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to parse metrics: %w", err)
	}

	metricTypes := r.GetMetricTypes()

	for _, metric := range parsedMetrics {
		name := metric.GetName()
		if metricTypes[name] {
			metricVal := metric.GetMetric()
			if len(metricVal) != 1 {
				r.log.Warn("expected 1 metric value", "got", len(metricVal), "metric", name)
			}
			if len(metricVal) > 0 {
				err = m.UpdatePrometheusMetric(name, metricVal[0])
				if err != nil {
					r.log.Warn("failed to add metric", "name", name, "error", err)
				}
			}
		}
	}

	r.metrics = append(r.metrics, *m.Copy())
	return nil
}
