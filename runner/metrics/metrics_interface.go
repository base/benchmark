package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"
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

func NewBlockMetrics(blockNumber uint64) *BlockMetrics {
	return &BlockMetrics{
		BlockNumber:      blockNumber,
		ExecutionMetrics: make(map[string]interface{}),
		Timestamp:        time.Now(),
	}
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
