package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type MetricsCollector interface {
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

func NewMetricsCollector(
	log log.Logger,
	client *ethclient.Client,
	clientName string,
	metricsPort int) MetricsCollector {
	switch clientName {
	case "geth":
		return NewGethMetricsCollector(log, client, metricsPort)
	case "reth":
		return NewRethMetricsCollector(log, client, metricsPort)
	}
	panic(fmt.Sprintf("unknown client: %s", clientName))
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
