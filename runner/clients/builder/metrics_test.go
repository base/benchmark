package builder

import (
	"math"
	"testing"

	io_prometheus_client "github.com/prometheus/client_model/go"
)

func TestPrometheusMetricNameSortsAndSanitizesLabels(t *testing.T) {
	metric := &io_prometheus_client.Metric{
		Label: []*io_prometheus_client.LabelPair{
			{Name: stringPtr("type"), Value: stringPtr("account.worker")},
			{Name: stringPtr("configname"), Value: stringPtr("mainnet/snapshot")},
		},
	}

	got := prometheusMetricName("reth_example_metric", metric)
	want := "reth_example_metric_configname_mainnet_snapshot_type_account_worker"
	if got != want {
		t.Fatalf("prometheusMetricName() = %q, want %q", got, want)
	}
}

func TestHistogramQuantileUsesIntervalBuckets(t *testing.T) {
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

	got, ok := histogramQuantile(0.5, current.Histogram, prev.Histogram)
	if !ok {
		t.Fatal("histogramQuantile() did not return a value")
	}
	if got != 2 {
		t.Fatalf("histogramQuantile(0.5) = %f, want 2", got)
	}

	got, ok = histogramQuantile(0.9, current.Histogram, prev.Histogram)
	if !ok {
		t.Fatal("histogramQuantile() did not return a value")
	}
	if got != 3 {
		t.Fatalf("histogramQuantile(0.9) = %f, want 3", got)
	}
}

func TestHistogramQuantileAvoidsInfiniteUpperBound(t *testing.T) {
	current := histogramMetric(
		10,
		bucket(1, 5),
		bucket(math.Inf(1), 10),
	)

	got, ok := histogramQuantile(0.9, current.Histogram, nil)
	if !ok {
		t.Fatal("histogramQuantile() did not return a value")
	}
	if got != 1 {
		t.Fatalf("histogramQuantile(0.9) = %f, want 1", got)
	}
}

func histogramMetric(count uint64, buckets ...*io_prometheus_client.Bucket) *io_prometheus_client.Metric {
	return &io_prometheus_client.Metric{
		Histogram: &io_prometheus_client.Histogram{
			SampleCount: uint64Ptr(count),
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
