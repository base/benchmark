package builder

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/base/base-bench/runner/metrics"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

type metricsCollector struct {
	log         log.Logger
	client      *ethclient.Client
	metrics     []metrics.BlockMetrics
	metricsPort int
	prevMetrics map[string]*io_prometheus_client.Metric
}

func newMetricsCollector(log log.Logger, client *ethclient.Client, metricsPort int) metrics.Collector {
	return &metricsCollector{
		log:         log,
		client:      client,
		metricsPort: metricsPort,
		metrics:     make([]metrics.BlockMetrics, 0),
		prevMetrics: make(map[string]*io_prometheus_client.Metric),
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
		"reth_sync_execution_execution_duration":                         true,
		"reth_sync_block_validation_state_root_duration":                 true,
		"reth_op_rbuilder_block_built_success":                           true,
		"reth_op_rbuilder_flashblock_count":                              true,
		"reth_op_rbuilder_total_block_built_duration":                    true,
		"reth_op_rbuilder_flashblock_build_duration":                     true,
		"reth_op_rbuilder_state_root_calculation_duration":               true,
		"reth_op_rbuilder_sequencer_tx_duration":                         true,
		"reth_op_rbuilder_payload_tx_simulation_duration":                true,
		"reth_base_builder_block_built_success":                          true,
		"reth_base_builder_flashblock_count":                             true,
		"reth_base_builder_total_block_built_duration":                   true,
		"reth_base_builder_flashblock_build_duration":                    true,
		"reth_base_builder_state_root_calculation_duration":              true,
		"reth_base_builder_state_root_time_per_gas_ratio":                true,
		"reth_base_builder_sequencer_tx_duration":                        true,
		"reth_base_builder_payload_transaction_simulation_duration":      true,
		"reth_base_builder_payload_tx_simulation_duration":               true,
		"reth_base_builder_tx_simulation_duration":                       true,
		"reth_base_builder_payload_num_tx_gauge":                         true,
		"reth_base_builder_flashblock_gas_headroom_pct":                  true,
		"reth_storage_providers_database_save_blocks_total":              true,
		"reth_storage_providers_database_save_blocks_block_count_last":   true,
		"reth_storage_providers_database_save_blocks_commit_sf":          true,
		"reth_storage_providers_database_save_blocks_commit_mdbx":        true,
		"reth_storage_providers_database_save_blocks_write_state":        true,
		"reth_storage_providers_database_save_blocks_write_hashed_state": true,
		"reth_storage_providers_database_save_blocks_write_trie_updates": true,
		"reth_storage_providers_database_save_blocks_sf":                 true,
		"reth_trie_leaves_added":                                         true,
		"reth_trie_branches_added":                                       true,
		"reth_tree_root_sparse_trie_total_duration_histogram":            true,
		"reth_tree_root_sparse_trie_final_update_duration_histogram":     true,
		"reth_parallel_sparse_trie_subtrie_hash_update_latency":          true,
		"reth_parallel_sparse_trie_subtrie_upper_hash_latency":           true,
		"reth_trie_proof_task_storage_worker_idle_time_seconds":          true,
		"reth_trie_proof_task_account_worker_idle_time_seconds":          true,
		"reth_trie_proof_task_blinded_storage_nodes":                     true,
		"reth_trie_proof_task_blinded_account_nodes":                     true,
		"reth_trie_cursor_overall_duration":                              true,
		"reth_trie_hashed_cursor_overall_duration":                       true,
		"reth_db_freelist": true,
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
	metrics, err := txtParser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to parse metrics: %w", err)
	}

	metricTypes := r.GetMetricTypes()
	m.SetPreviousPrometheusMetrics(r.prevMetrics)

	for _, metric := range metrics {
		name := metric.GetName()
		if metricTypes[name] {
			metricVal := metric.GetMetric()
			for _, value := range metricVal {
				metricName := prometheusMetricName(name, value)
				addPrometheusMetric(r.log, m, metricName, value)

				if metricName != name && len(metricVal) == 1 {
					addPrometheusMetric(r.log, m, name, value)
				}
			}
		}
	}

	r.prevMetrics = m.PreviousPrometheusMetrics()
	r.metrics = append(r.metrics, *m.Copy())
	return nil
}

func addPrometheusMetric(log log.Logger, m *metrics.BlockMetrics, name string, metric *io_prometheus_client.Metric) {
	addHistogramQuantiles(m, name, metric, m.PreviousPrometheusMetric(name))

	err := m.UpdatePrometheusMetric(name, metric)
	if err != nil {
		log.Warn("failed to add metric %s: %s", name, err)
	}
	addSummaryQuantiles(m, name, metric)
}

func prometheusMetricName(name string, metric *io_prometheus_client.Metric) string {
	labels := metric.GetLabel()
	if len(labels) == 0 {
		return name
	}

	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		parts = append(parts, sanitizeMetricPart(label.GetName())+"_"+sanitizeMetricPart(label.GetValue()))
	}
	sort.Strings(parts)
	return name + "_" + strings.Join(parts, "_")
}

func addSummaryQuantiles(m *metrics.BlockMetrics, name string, metric *io_prometheus_client.Metric) {
	if metric.Summary == nil {
		return
	}

	for _, quantile := range metric.Summary.GetQuantile() {
		m.AddExecutionMetric(name+"_quantile_"+formatQuantile(quantile.GetQuantile()), quantile.GetValue())
	}
}

func addHistogramQuantiles(m *metrics.BlockMetrics, name string, metric *io_prometheus_client.Metric, prevMetric *io_prometheus_client.Metric) {
	if metric.Histogram == nil {
		return
	}

	var prevHistogram *io_prometheus_client.Histogram
	if prevMetric != nil {
		prevHistogram = prevMetric.Histogram
	}

	for _, quantile := range []float64{0.5, 0.9, 0.99} {
		value, ok := histogramQuantile(quantile, metric.Histogram, prevHistogram)
		if ok {
			m.AddExecutionMetric(name+"_quantile_"+formatQuantile(quantile), value)
		}
	}
}

func histogramQuantile(quantile float64, histogram *io_prometheus_client.Histogram, prevHistogram *io_prometheus_client.Histogram) (float64, bool) {
	if histogram == nil || histogram.SampleCount == nil || len(histogram.Bucket) == 0 {
		return 0, false
	}

	prevCount := uint64(0)
	if prevHistogram != nil && prevHistogram.SampleCount != nil {
		prevCount = *prevHistogram.SampleCount
	}
	count := deltaUint64(*histogram.SampleCount, prevCount)
	if count == 0 {
		return 0, false
	}

	rank := quantile * float64(count)
	lastFiniteUpperBound := 0.0
	hasFiniteUpperBound := false
	for i, bucket := range histogram.Bucket {
		if bucket.CumulativeCount == nil || bucket.UpperBound == nil {
			continue
		}
		if !math.IsInf(*bucket.UpperBound, 0) {
			lastFiniteUpperBound = *bucket.UpperBound
			hasFiniteUpperBound = true
		}

		prevBucketCount := uint64(0)
		if prevHistogram != nil && i < len(prevHistogram.Bucket) && prevHistogram.Bucket[i].CumulativeCount != nil {
			prevBucketCount = *prevHistogram.Bucket[i].CumulativeCount
		}

		bucketCount := deltaUint64(*bucket.CumulativeCount, prevBucketCount)
		if float64(bucketCount) >= rank {
			if math.IsInf(*bucket.UpperBound, 0) {
				return lastFiniteUpperBound, hasFiniteUpperBound
			}
			return *bucket.UpperBound, true
		}
	}

	return 0, false
}

func deltaUint64(current uint64, previous uint64) uint64 {
	if current < previous {
		return current
	}
	return current - previous
}

func formatQuantile(quantile float64) string {
	return strings.ReplaceAll(strconv.FormatFloat(quantile, 'f', -1, 64), ".", "_")
}

func sanitizeMetricPart(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, value)
}
