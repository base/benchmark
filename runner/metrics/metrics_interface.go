package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type MetricsCollector interface {
	Collect(ctx context.Context) error
	GetMetricsEndpoint() string
	GetMetrics() []Metrics
}

type Metrics struct {
	BlockNumber      uint64
	Timestamp        time.Time
	ExecutionMetrics map[string]interface{}
}

func NewMetrics() *Metrics {
	return &Metrics{
		ExecutionMetrics: make(map[string]interface{}),
		Timestamp:        time.Now(),
	}
}

func (m *Metrics) AddExecutionMetric(name string, value interface{}) {
	m.ExecutionMetrics[name] = value
}

func (m *Metrics) GetMetricTypes() map[string]bool {
	return map[string]bool{
		"execution": true,
	}
}

func NewMetricsCollector(
	log log.Logger,
	client *ethclient.Client,
	clientName string) MetricsCollector {
	switch clientName {
	case "geth":
		return NewGethMetricsCollector(log, client)
	case "reth":
		return NewRethMetricsCollector(log, client)
	}
	panic(fmt.Sprintf("unknown client: %s", clientName))
}

type MetricsWriter interface {
	Write(metrics []Metrics) error
}

type FileMetricsWriter struct {
	BaseDir string
}

func NewFileMetricsWriter(baseDir string) *FileMetricsWriter {
	return &FileMetricsWriter{
		BaseDir: baseDir,
	}
}

func (w *FileMetricsWriter) Write(metrics []Metrics) error {
	timestamp := time.Now().Format("20060102_150405")
	filename := w.BaseDir + "/metrics_" + timestamp + ".json"

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	return nil
}
