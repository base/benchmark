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
		"reth_base_builder_transaction_pool_fetch_duration":              true,
		"reth_base_builder_transaction_pool_fetch_gauge":                 true,
		"reth_base_builder_state_transition_merge_duration":              true,
		"reth_base_builder_state_transition_merge_gauge":                 true,
		"reth_base_builder_payload_num_tx_considered":                    true,
		"reth_base_builder_payload_num_tx_considered_gauge":              true,
		"reth_base_builder_payload_num_tx":                               true,
		"reth_base_builder_payload_num_tx_gauge":                         true,
		"reth_base_builder_payload_num_tx_simulated":                     true,
		"reth_base_builder_payload_num_tx_simulated_gauge":               true,
		"reth_base_builder_payload_num_tx_simulated_success":             true,
		"reth_base_builder_payload_num_tx_simulated_success_gauge":       true,
		"reth_base_builder_payload_num_tx_simulated_fail":                true,
		"reth_base_builder_payload_num_tx_simulated_fail_gauge":          true,
		"reth_base_builder_payload_reverted_tx_gas_used":                 true,
		"reth_base_builder_reverted_tx_gas_used":                         true,
		"reth_base_builder_successful_tx_gas_used":                       true,
		"reth_base_builder_tx_accounts_modified":                         true,
		"reth_base_builder_tx_storage_slots_modified":                    true,
		"reth_base_builder_rejection_cache_hits":                         true,
		"reth_base_builder_rejection_cache_insertions":                   true,
		"reth_base_builder_rejection_cache_size":                         true,
		"reth_base_builder_metering_data_pending_skip":                   true,
		"reth_base_builder_gas_limit_exceeded_total":                     true,
		"reth_base_builder_tx_da_size_exceeded_total":                    true,
		"reth_base_builder_block_da_size_exceeded_total":                 true,
		"reth_base_builder_da_footprint_exceeded_total":                  true,
		"reth_base_builder_block_uncompressed_size_exceeded_total":       true,
		"reth_base_builder_block_uncompressed_size":                      true,
		"reth_base_builder_resource_limit_would_reject_total":            true,
		"reth_base_builder_tx_execution_time_exceeded_total":             true,
		"reth_base_builder_flashblock_execution_time_exceeded_total":     true,
		"reth_base_builder_block_state_root_gas_exceeded_total":          true,
		"reth_base_builder_flashblock_txs_considered":                    true,
		"reth_base_builder_flashblock_txs_included":                      true,
		"reth_base_builder_flashblock_txs_rejected":                      true,
		"reth_base_builder_flashblock_selection_total":                   true,
		"reth_base_builder_flashblock_rejections_total":                  true,
		"reth_base_builder_flashblock_gas_headroom":                      true,
		"reth_base_builder_flashblock_gas_headroom_pct":                  true,
		"reth_base_builder_flashblock_da_bytes_used":                     true,
		"reth_base_builder_flashblock_da_headroom_bytes":                 true,
		"reth_base_builder_flashblock_execution_time_used_us":            true,
		"reth_base_builder_flashblock_execution_time_headroom_us":        true,
		"reth_base_builder_flashblock_state_root_gas_used":               true,
		"reth_base_builder_flashblock_state_root_gas_headroom":           true,
		"reth_base_builder_flashblock_byte_size_histogram":               true,
		"reth_base_builder_flashblock_num_tx_histogram":                  true,
		"reth_base_builder_payload_byte_size":                            true,
		"reth_base_builder_payload_byte_size_gauge":                      true,
		"reth_base_builder_tx_byte_size":                                 true,
		"reth_base_builder_block_state_root_gas":                         true,
		"reth_base_builder_flashblocks_time_drift":                       true,
		"reth_base_builder_first_flashblock_time_offset":                 true,
		"reth_base_builder_reduced_flashblocks_number":                   true,
		"reth_base_builder_missing_flashblocks_count":                    true,
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
				metricName := r.prometheusMetricName(name, value)
				r.addPrometheusMetric(m, name, metricName, value, true)

				if metricName != name && len(metricVal) == 1 {
					r.addPrometheusMetric(m, name, name, value, false)
				}
			}
		}
	}

	r.prevMetrics = m.PreviousPrometheusMetrics()
	r.metrics = append(r.metrics, *m.Copy())
	return nil
}

func (r *metricsCollector) addPrometheusMetric(m *metrics.BlockMetrics, rawName string, name string, metric *io_prometheus_client.Metric, recordSample bool) {
	prevMetric := m.PreviousPrometheusMetric(name)
	r.addHistogramQuantiles(m, name, metric, prevMetric)
	if recordSample {
		m.AddPrometheusMetricSample(r.prometheusMetricSample(rawName, name, metric, prevMetric))
	}

	err := m.UpdatePrometheusMetric(name, metric)
	if err != nil {
		r.log.Warn("failed to add metric %s: %s", name, err)
	}
	r.addSummaryQuantiles(m, name, metric)
}

func (r *metricsCollector) prometheusMetricSample(rawName string, key string, metric *io_prometheus_client.Metric, prevMetric *io_prometheus_client.Metric) metrics.PrometheusMetricSample {
	sample := metrics.PrometheusMetricSample{
		Name:   rawName,
		Key:    key,
		Labels: r.prometheusLabels(metric),
	}

	if metric.Gauge != nil {
		sample.Type = "gauge"
		if metric.Gauge.Value != nil && !math.IsNaN(*metric.Gauge.Value) {
			sample.Value = prometheusFloat64Ptr(*metric.Gauge.Value)
		}
		return sample
	}

	if metric.Counter != nil {
		sample.Type = "counter"
		if metric.Counter.Value != nil && !math.IsNaN(*metric.Counter.Value) {
			value := *metric.Counter.Value
			sample.Value = prometheusFloat64Ptr(value)
			if prevMetric != nil && prevMetric.Counter != nil && prevMetric.Counter.Value != nil {
				sample.Delta = prometheusFloat64Ptr(r.deltaFloat64(value, *prevMetric.Counter.Value))
			} else {
				sample.Delta = prometheusFloat64Ptr(value)
			}
		}
		return sample
	}

	if metric.Histogram != nil {
		sample.Type = "histogram"
		histogram := metric.Histogram
		if histogram.SampleSum != nil && !math.IsNaN(*histogram.SampleSum) {
			sum := *histogram.SampleSum
			sample.Sum = prometheusFloat64Ptr(sum)
			if prevMetric != nil && prevMetric.Histogram != nil && prevMetric.Histogram.SampleSum != nil {
				sample.SumDelta = prometheusFloat64Ptr(r.deltaFloat64(sum, *prevMetric.Histogram.SampleSum))
			} else {
				sample.SumDelta = prometheusFloat64Ptr(sum)
			}
		}
		if histogram.SampleCount != nil {
			count := *histogram.SampleCount
			sample.Count = prometheusUint64Ptr(count)
			if prevMetric != nil && prevMetric.Histogram != nil && prevMetric.Histogram.SampleCount != nil {
				countDelta := r.deltaUint64(count, *prevMetric.Histogram.SampleCount)
				sample.CountDelta = &countDelta
			} else {
				sample.CountDelta = prometheusUint64Ptr(count)
			}
		}
		return sample
	}

	if metric.Summary != nil {
		sample.Type = "summary"
		summary := metric.Summary
		if summary.SampleSum != nil && !math.IsNaN(*summary.SampleSum) {
			sum := *summary.SampleSum
			sample.Sum = prometheusFloat64Ptr(sum)
			if prevMetric != nil && prevMetric.Summary != nil && prevMetric.Summary.SampleSum != nil {
				sample.SumDelta = prometheusFloat64Ptr(r.deltaFloat64(sum, *prevMetric.Summary.SampleSum))
			} else {
				sample.SumDelta = prometheusFloat64Ptr(sum)
			}
		}
		if summary.SampleCount != nil {
			count := *summary.SampleCount
			sample.Count = prometheusUint64Ptr(count)
			if prevMetric != nil && prevMetric.Summary != nil && prevMetric.Summary.SampleCount != nil {
				countDelta := r.deltaUint64(count, *prevMetric.Summary.SampleCount)
				sample.CountDelta = &countDelta
			} else {
				sample.CountDelta = prometheusUint64Ptr(count)
			}
		}
		if len(summary.Quantile) > 0 {
			sample.Quantiles = make(map[string]float64, len(summary.Quantile))
			for _, quantile := range summary.Quantile {
				sample.Quantiles[r.formatQuantile(quantile.GetQuantile())] = quantile.GetValue()
			}
		}
		return sample
	}

	sample.Type = "unknown"
	return sample
}

func (r *metricsCollector) prometheusMetricName(name string, metric *io_prometheus_client.Metric) string {
	labels := metric.GetLabel()
	if len(labels) == 0 {
		return name
	}

	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		parts = append(parts, r.sanitizeMetricPart(label.GetName())+"_"+r.sanitizeMetricPart(label.GetValue()))
	}
	sort.Strings(parts)
	return name + "_" + strings.Join(parts, "_")
}

func (r *metricsCollector) prometheusLabels(metric *io_prometheus_client.Metric) map[string]string {
	labels := metric.GetLabel()
	if len(labels) == 0 {
		return nil
	}

	result := make(map[string]string, len(labels))
	for _, label := range labels {
		result[label.GetName()] = label.GetValue()
	}
	return result
}

func (r *metricsCollector) addSummaryQuantiles(m *metrics.BlockMetrics, name string, metric *io_prometheus_client.Metric) {
	if metric.Summary == nil {
		return
	}

	for _, quantile := range metric.Summary.GetQuantile() {
		m.AddExecutionMetric(name+"_quantile_"+r.formatQuantile(quantile.GetQuantile()), quantile.GetValue())
	}
}

func (r *metricsCollector) addHistogramQuantiles(m *metrics.BlockMetrics, name string, metric *io_prometheus_client.Metric, prevMetric *io_prometheus_client.Metric) {
	if metric.Histogram == nil {
		return
	}

	var prevHistogram *io_prometheus_client.Histogram
	if prevMetric != nil {
		prevHistogram = prevMetric.Histogram
	}

	for _, quantile := range []float64{0.5, 0.9, 0.99} {
		value, ok := r.histogramQuantile(quantile, metric.Histogram, prevHistogram)
		if ok {
			m.AddExecutionMetric(name+"_quantile_"+r.formatQuantile(quantile), value)
		}
	}
}

func (r *metricsCollector) histogramQuantile(quantile float64, histogram *io_prometheus_client.Histogram, prevHistogram *io_prometheus_client.Histogram) (float64, bool) {
	if histogram == nil || histogram.SampleCount == nil || len(histogram.Bucket) == 0 {
		return 0, false
	}

	prevCount := uint64(0)
	if prevHistogram != nil && prevHistogram.SampleCount != nil {
		prevCount = *prevHistogram.SampleCount
	}
	count := r.deltaUint64(*histogram.SampleCount, prevCount)
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

		bucketCount := r.deltaUint64(*bucket.CumulativeCount, prevBucketCount)
		if float64(bucketCount) >= rank {
			if math.IsInf(*bucket.UpperBound, 0) {
				return lastFiniteUpperBound, hasFiniteUpperBound
			}
			return *bucket.UpperBound, true
		}
	}

	return 0, false
}

func (r *metricsCollector) deltaUint64(current uint64, previous uint64) uint64 {
	if current < previous {
		return current
	}
	return current - previous
}

func (r *metricsCollector) deltaFloat64(current float64, previous float64) float64 {
	if current < previous {
		return current
	}
	return current - previous
}

func (r *metricsCollector) formatQuantile(quantile float64) string {
	return strings.ReplaceAll(strconv.FormatFloat(quantile, 'f', -1, 64), ".", "_")
}

func prometheusFloat64Ptr(value float64) *float64 {
	return &value
}

func prometheusUint64Ptr(value uint64) *uint64 {
	return &value
}

func (r *metricsCollector) sanitizeMetricPart(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, value)
}
