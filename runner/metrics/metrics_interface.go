package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"os"
	"path"
	"time"

	io_prometheus_client "github.com/prometheus/client_model/go"
)

type Collector interface {
	Collect(ctx context.Context, metrics *BlockMetrics) error
	GetMetrics() []BlockMetrics
}

type BlockMetrics struct {
	BlockNumber      uint64
	Timestamp        time.Time
	ExecutionMetrics map[string]interface{}
}

func NewBlockMetrics() *BlockMetrics {
	return &BlockMetrics{
		BlockNumber:      0,
		ExecutionMetrics: make(map[string]interface{}),
		Timestamp:        time.Now(),
	}
}

func (m *BlockMetrics) SetBlockNumber(blockNumber uint64) {
	m.BlockNumber = blockNumber
}

func (m *BlockMetrics) Copy() *BlockMetrics {
	newMetrics := make(map[string]interface{})
	maps.Copy(newMetrics, m.ExecutionMetrics)
	return &BlockMetrics{
		BlockNumber:      m.BlockNumber,
		ExecutionMetrics: newMetrics,
		Timestamp:        m.Timestamp,
	}
}

func (m *BlockMetrics) UpdatePrometheusMetric(name string, value *io_prometheus_client.Metric) error {
	if value.Histogram != nil {
		// get the average change in sum divided by the average change in count
		prevSum := 0.0
		prevValue, ok := m.ExecutionMetrics[name].(*io_prometheus_client.Metric)
		if !ok {
			prevValue = nil
		}
		if prevValue != nil {
			if prevValue.Histogram.SampleSum != nil {
				prevSum = *prevValue.Histogram.SampleSum
			}
		}
		sum := 0.0
		if value.Histogram.SampleSum != nil {
			sum = *value.Histogram.SampleSum
		}
		prevCount := 0.0
		if prevValue != nil {
			if prevValue.Histogram.SampleCount != nil {
				prevCount = float64(*prevValue.Histogram.SampleCount)
			}
		}
		count := 0.0
		if value.Histogram.SampleCount != nil {
			count = float64(*value.Histogram.SampleCount)
		}
		deltaCount := count - prevCount
		if deltaCount != 0 {
			averageChange := (sum - prevSum) / deltaCount

			if !math.IsNaN(averageChange) {
				m.ExecutionMetrics[name] = averageChange
			}
		}
	} else if value.Gauge != nil {
		if value.Gauge.Value != nil && !math.IsNaN(*value.Gauge.Value) {
			m.ExecutionMetrics[name] = *value.Gauge.Value
		}
		// NaN values and nil values are silently omitted
	} else if value.Counter != nil {
		if value.Counter.Value != nil && !math.IsNaN(*value.Counter.Value) {
			m.ExecutionMetrics[name] = *value.Counter.Value
		}
		// NaN values and nil values are silently omitted
	} else if value.Summary != nil {
		// get the average change in sum divided by the average change in count
		prevSum := 0.0

		prevValue, ok := m.ExecutionMetrics[name].(*io_prometheus_client.Metric)
		if !ok {
			prevValue = nil
		}
		if prevValue != nil {
			if prevValue.Summary.SampleSum != nil {
				prevSum = *prevValue.Summary.SampleSum
			}
		}
		sum := 0.0
		if value.Summary.SampleSum != nil {
			sum = *value.Summary.SampleSum
		}
		prevCount := 0.0
		if prevValue != nil {
			if prevValue.Summary.SampleCount != nil {
				prevCount = float64(*prevValue.Summary.SampleCount)
			}
		}
		count := 0.0
		if value.Summary.SampleCount != nil {
			count = float64(*value.Summary.SampleCount)
		}
		denom := count - prevCount
		if denom != 0 {
			averageChange := (sum - prevSum) / denom
			if !math.IsNaN(averageChange) {
				m.ExecutionMetrics[name] = averageChange
			}
		}

	} else {
		return fmt.Errorf("invalid metric type for %s: %#v", name, value)
	}
	return nil
}
func (m *BlockMetrics) AddExecutionMetric(name string, value interface{}) {
	m.ExecutionMetrics[name] = value
}

func (m *BlockMetrics) GetMetricTypes() map[string]bool {
	return map[string]bool{
		"execution": true,
	}
}

func (m *BlockMetrics) GetMetricFloat(name string) (float64, bool) {
	if value, ok := m.ExecutionMetrics[name]; ok {
		if v, ok := value.(time.Time); ok {
			return float64(v.UnixNano()) / 1e9, true
		} else if v, ok := value.(time.Duration); ok {
			return float64(v.Nanoseconds()) / 1e9, true
		}
		switch v := value.(type) {
		case float64:
			return v, true
		case float32:
			return float64(v), true
		case int:
			return float64(v), true
		case int64:
			return float64(v), true
		case uint:
			return float64(v), true
		case uint64:
			return float64(v), true
		}
	}

	return 0, false
}

type MetricsWriter interface {
	Write(metrics []BlockMetrics) error
}

type FileMetricsWriter struct {
	BaseDir string
}

func NewFileMetricsWriter(baseDir string) *FileMetricsWriter {
	return &FileMetricsWriter{
		BaseDir: baseDir,
	}
}

const MetricsFileName = "metrics.json"

func (w *FileMetricsWriter) Write(metrics []BlockMetrics) error {
	fmt.Println("Writing metrics to", w.BaseDir)
	filename := path.Join(w.BaseDir, MetricsFileName)

	data, err := json.MarshalIndent(metrics, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	return nil
}
