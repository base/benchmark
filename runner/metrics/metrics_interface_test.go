package metrics

import (
	"math"
	"testing"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestBlockMetrics_UpdatePrometheusMetric_NaNHandling(t *testing.T) {
	tests := []struct {
		name           string
		metricName     string
		metric         *io_prometheus_client.Metric
		expectedMetric string
		shouldContain  bool
		description    string
	}{
		{
			name:       "gauge with NaN value should be omitted",
			metricName: "test_gauge",
			metric: &io_prometheus_client.Metric{
				Gauge: &io_prometheus_client.Gauge{
					Value: floatPtr(math.NaN()),
				},
			},
			expectedMetric: "test_gauge",
			shouldContain:  false,
			description:    "NaN gauge values should not be added to ExecutionMetrics",
		},
		{
			name:       "gauge with valid value should be included",
			metricName: "test_gauge",
			metric: &io_prometheus_client.Metric{
				Gauge: &io_prometheus_client.Gauge{
					Value: floatPtr(42.5),
				},
			},
			expectedMetric: "test_gauge",
			shouldContain:  true,
			description:    "Valid gauge values should be added to ExecutionMetrics",
		},
		{
			name:       "counter with NaN value should be omitted",
			metricName: "test_counter",
			metric: &io_prometheus_client.Metric{
				Counter: &io_prometheus_client.Counter{
					Value: floatPtr(math.NaN()),
				},
			},
			expectedMetric: "test_counter",
			shouldContain:  false,
			description:    "NaN counter values should not be added to ExecutionMetrics",
		},
		{
			name:       "counter with valid value should be included",
			metricName: "test_counter",
			metric: &io_prometheus_client.Metric{
				Counter: &io_prometheus_client.Counter{
					Value: floatPtr(100.0),
				},
			},
			expectedMetric: "test_counter",
			shouldContain:  true,
			description:    "Valid counter values should be added to ExecutionMetrics",
		},
		{
			name:       "histogram with NaN average should omit _avg metric",
			metricName: "test_histogram",
			metric: &io_prometheus_client.Metric{
				Histogram: &io_prometheus_client.Histogram{
					SampleSum:   floatPtr(math.NaN()),
					SampleCount: uint64Ptr(10),
				},
			},
			expectedMetric: "test_histogram_avg",
			shouldContain:  false,
			description:    "Histogram with NaN sum should not create _avg metric",
		},
		{
			name:       "summary with NaN average should omit _avg metric",
			metricName: "test_summary",
			metric: &io_prometheus_client.Metric{
				Summary: &io_prometheus_client.Summary{
					SampleSum:   floatPtr(math.NaN()),
					SampleCount: uint64Ptr(5),
				},
			},
			expectedMetric: "test_summary_avg",
			shouldContain:  false,
			description:    "Summary with NaN sum should not create _avg metric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewBlockMetrics()

			err := m.UpdatePrometheusMetric(tt.metricName, tt.metric)
			require.NoError(t, err, "UpdatePrometheusMetric should not return error")

			_, exists := m.ExecutionMetrics[tt.expectedMetric]
			if tt.shouldContain {
				require.True(t, exists, tt.description)
			} else {
				require.False(t, exists, tt.description)
			}
		})
	}
}

func TestBlockMetrics_UpdatePrometheusMetric_HistogramAverage(t *testing.T) {
	m := NewBlockMetrics()

	// First update - establish baseline
	err := m.UpdatePrometheusMetric("test_histogram", &io_prometheus_client.Metric{
		Histogram: &io_prometheus_client.Histogram{
			SampleSum:   floatPtr(100.0),
			SampleCount: uint64Ptr(10),
		},
	})
	require.NoError(t, err)

	// Store the current metric for the next calculation
	m.ExecutionMetrics["test_histogram"] = &io_prometheus_client.Metric{
		Histogram: &io_prometheus_client.Histogram{
			SampleSum:   floatPtr(100.0),
			SampleCount: uint64Ptr(10),
		},
	}

	// Second update - should calculate average change
	err = m.UpdatePrometheusMetric("test_histogram", &io_prometheus_client.Metric{
		Histogram: &io_prometheus_client.Histogram{
			SampleSum:   floatPtr(150.0),
			SampleCount: uint64Ptr(15),
		},
	})
	require.NoError(t, err)

	// Should have calculated average: (150-100)/(15-10) = 50/5 = 10
	avgValue, exists := m.ExecutionMetrics["test_histogram_avg"]
	require.True(t, exists, "Average metric should be created")
	require.Equal(t, 10.0, avgValue, "Average should be calculated correctly")
}

func TestBlockMetrics_UpdatePrometheusMetric_SummaryAverage(t *testing.T) {
	m := NewBlockMetrics()

	// First update - establish baseline
	err := m.UpdatePrometheusMetric("test_summary", &io_prometheus_client.Metric{
		Summary: &io_prometheus_client.Summary{
			SampleSum:   floatPtr(200.0),
			SampleCount: uint64Ptr(20),
		},
	})
	require.NoError(t, err)

	// Store the current metric for the next calculation
	m.ExecutionMetrics["test_summary"] = &io_prometheus_client.Metric{
		Summary: &io_prometheus_client.Summary{
			SampleSum:   floatPtr(200.0),
			SampleCount: uint64Ptr(20),
		},
	}

	// Second update - should calculate average change
	err = m.UpdatePrometheusMetric("test_summary", &io_prometheus_client.Metric{
		Summary: &io_prometheus_client.Summary{
			SampleSum:   floatPtr(300.0),
			SampleCount: uint64Ptr(25),
		},
	})
	require.NoError(t, err)

	// Should have calculated average: (300-200)/(25-20) = 100/5 = 20
	avgValue, exists := m.ExecutionMetrics["test_summary_avg"]
	require.True(t, exists, "Average metric should be created")
	require.Equal(t, 20.0, avgValue, "Average should be calculated correctly")
}

func TestBlockMetrics_UpdatePrometheusMetric_ZeroDenominator(t *testing.T) {
	m := NewBlockMetrics()

	// Store a baseline metric
	m.ExecutionMetrics["test_histogram"] = &io_prometheus_client.Metric{
		Histogram: &io_prometheus_client.Histogram{
			SampleSum:   floatPtr(100.0),
			SampleCount: uint64Ptr(10),
		},
	}

	// Update with same count (zero denominator)
	err := m.UpdatePrometheusMetric("test_histogram", &io_prometheus_client.Metric{
		Histogram: &io_prometheus_client.Histogram{
			SampleSum:   floatPtr(120.0),
			SampleCount: uint64Ptr(10), // Same count = zero denominator
		},
	})
	require.NoError(t, err)

	// Should not create _avg metric when denominator is zero
	_, exists := m.ExecutionMetrics["test_histogram_avg"]
	require.False(t, exists, "Average metric should not be created with zero denominator")
}

func TestBlockMetrics_UpdatePrometheusMetric_InvalidMetricType(t *testing.T) {
	m := NewBlockMetrics()

	// Test with empty metric (no gauge, counter, histogram, or summary)
	err := m.UpdatePrometheusMetric("invalid_metric", &io_prometheus_client.Metric{})
	require.Error(t, err, "Should return error for invalid metric type")
	require.Contains(t, err.Error(), "invalid metric type", "Error should mention invalid metric type")
}

func TestBlockMetrics_UpdatePrometheusMetric_NilValues(t *testing.T) {
	m := NewBlockMetrics()

	// Test gauge with nil value (should not panic and should not add metric)
	err := m.UpdatePrometheusMetric("test_gauge_nil", &io_prometheus_client.Metric{
		Gauge: &io_prometheus_client.Gauge{
			Value: nil,
		},
	})
	require.NoError(t, err, "Should handle nil gauge value gracefully")

	_, exists := m.ExecutionMetrics["test_gauge_nil"]
	require.False(t, exists, "Should not add gauge metric with nil value")

	// Test counter with nil value (should not panic and should not add metric)
	err = m.UpdatePrometheusMetric("test_counter_nil", &io_prometheus_client.Metric{
		Counter: &io_prometheus_client.Counter{
			Value: nil,
		},
	})
	require.NoError(t, err, "Should handle nil counter value gracefully")

	_, exists = m.ExecutionMetrics["test_counter_nil"]
	require.False(t, exists, "Should not add counter metric with nil value")
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}

func uint64Ptr(u uint64) *uint64 {
	return &u
}
