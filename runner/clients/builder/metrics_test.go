package builder

import (
	"math"
	"testing"

	io_prometheus_client "github.com/prometheus/client_model/go"
)

func TestPrometheusMetricNameSortsAndSanitizesLabels(t *testing.T) {
	collector := &metricsCollector{}
	metric := &io_prometheus_client.Metric{
		Label: []*io_prometheus_client.LabelPair{
			{Name: stringPtr("type"), Value: stringPtr("account.worker")},
			{Name: stringPtr("configname"), Value: stringPtr("mainnet/snapshot")},
		},
	}

	got := collector.prometheusMetricName("reth_example_metric", metric)
	want := "reth_example_metric_configname_mainnet_snapshot_type_account_worker"
	if got != want {
		t.Fatalf("prometheusMetricName() = %q, want %q", got, want)
	}
}

func TestHistogramQuantileUsesIntervalBuckets(t *testing.T) {
	collector := &metricsCollector{}
	prev := histogramMetric(
		10,
		bucket(1, 5),
		bucket(2, 9),
		bucket(3, 10),
	)
	current := histogramMetric(
		20,
		bucket(1, 8),
		bucket(2, 17),
		bucket(3, 20),
	)

	got, ok := collector.histogramQuantile(0.5, current.Histogram, prev.Histogram)
	if !ok {
		t.Fatal("histogramQuantile() did not return a value")
	}
	if got != 2 {
		t.Fatalf("histogramQuantile(0.5) = %f, want 2", got)
	}

	got, ok = collector.histogramQuantile(0.9, current.Histogram, prev.Histogram)
	if !ok {
		t.Fatal("histogramQuantile() did not return a value")
	}
	if got != 3 {
		t.Fatalf("histogramQuantile(0.9) = %f, want 3", got)
	}
}

func TestHistogramQuantileAvoidsInfiniteUpperBound(t *testing.T) {
	collector := &metricsCollector{}
	current := histogramMetric(
		10,
		bucket(1, 5),
		bucket(math.Inf(1), 10),
	)

	got, ok := collector.histogramQuantile(0.9, current.Histogram, nil)
	if !ok {
		t.Fatal("histogramQuantile() did not return a value")
	}
	if got != 1 {
		t.Fatalf("histogramQuantile(0.9) = %f, want 1", got)
	}
}

func TestPrometheusMetricSamplePreservesLabelsAndHistogramDeltas(t *testing.T) {
	collector := &metricsCollector{}
	prev := histogramMetric(
		10,
		bucket(1, 5),
		bucket(2, 10),
	)
	*prev.Histogram.SampleSum = 100

	current := histogramMetric(
		16,
		bucket(1, 8),
		bucket(2, 16),
	)
	*current.Histogram.SampleSum = 172
	current.Label = []*io_prometheus_client.LabelPair{
		{Name: stringPtr("flashblock_index"), Value: stringPtr("7")},
	}

	sample := collector.prometheusMetricSample(
		"reth_base_builder_flashblock_txs_considered",
		"reth_base_builder_flashblock_txs_considered_flashblock_index_7",
		current,
		prev,
	)

	if sample.Name != "reth_base_builder_flashblock_txs_considered" {
		t.Fatalf("Name = %q", sample.Name)
	}
	if sample.Key != "reth_base_builder_flashblock_txs_considered_flashblock_index_7" {
		t.Fatalf("Key = %q", sample.Key)
	}
	if sample.Labels["flashblock_index"] != "7" {
		t.Fatalf("flashblock_index label = %q", sample.Labels["flashblock_index"])
	}
	if sample.Count == nil || *sample.Count != 16 {
		t.Fatalf("Count = %v, want 16", sample.Count)
	}
	if sample.CountDelta == nil || *sample.CountDelta != 6 {
		t.Fatalf("CountDelta = %v, want 6", sample.CountDelta)
	}
	if sample.Sum == nil || *sample.Sum != 172 {
		t.Fatalf("Sum = %v, want 172", sample.Sum)
	}
	if sample.SumDelta == nil || *sample.SumDelta != 72 {
		t.Fatalf("SumDelta = %v, want 72", sample.SumDelta)
	}
}

func TestBuilderMetricTypesIncludeFlashblockDiagnostics(t *testing.T) {
	collector := &metricsCollector{}
	metricTypes := collector.GetMetricTypes()

	for _, name := range []string{
		"reth_base_builder_flashblock_txs_considered",
		"reth_base_builder_flashblock_txs_included",
		"reth_base_builder_flashblock_txs_rejected",
		"reth_base_builder_flashblock_selection_total",
		"reth_base_builder_flashblock_rejections_total",
		"reth_base_builder_transaction_pool_fetch_duration",
		"reth_base_builder_state_transition_merge_duration",
		"reth_base_builder_flashblock_count",
	} {
		if !metricTypes[name] {
			t.Fatalf("GetMetricTypes()[%q] = false, want true", name)
		}
	}
}

func histogramMetric(count uint64, buckets ...*io_prometheus_client.Bucket) *io_prometheus_client.Metric {
	return &io_prometheus_client.Metric{
		Histogram: &io_prometheus_client.Histogram{
			SampleCount: uint64Ptr(count),
			SampleSum:   float64Ptr(0),
			Bucket:      buckets,
		},
	}
}

func bucket(upperBound float64, count uint64) *io_prometheus_client.Bucket {
	return &io_prometheus_client.Bucket{
		UpperBound:      float64Ptr(upperBound),
		CumulativeCount: uint64Ptr(count),
	}
}

func stringPtr(value string) *string {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}
