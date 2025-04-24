package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/base/base-bench/runner/benchmark"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	UpdateForkChoiceLatencyMetric = "latency/update_fork_choice"
	NewPayloadLatencyMetric       = "latency/new_payload"
	GetPayloadLatencyMetric       = "latency/get_payload"
	SendTxsLatencyMetric          = "latency/send_txs"
	GasPerBlockMetric             = "gas/per_block"
	GasPerSecondMetric            = "gas/per_second"
	TransactionsPerBlockMetric    = "transactions/per_block"
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

func getAverage(metrics []BlockMetrics, metricName string) float64 {
	var total float64
	var count int
	for _, metric := range metrics {
		if value, ok := metric.GetMetricFloat(metricName); ok {
			total += value
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func BlockMetricsToValidatorSummary(metrics []BlockMetrics) *benchmark.ValidatorKeyMetrics {
	averageNewPayloadLatency := getAverage(metrics, NewPayloadLatencyMetric)
	averageGasPerSecond := getAverage(metrics, GasPerSecondMetric)

	return &benchmark.ValidatorKeyMetrics{
		AverageNewPayloadLatency: averageNewPayloadLatency,
		CommonKeyMetrics: benchmark.CommonKeyMetrics{
			AverageGasPerSecond: averageGasPerSecond,
		},
	}
}

func BlockMetricsToSequencerSummary(metrics []BlockMetrics) *benchmark.SequencerKeyMetrics {
	averageUpdateForkChoiceLatency := getAverage(metrics, UpdateForkChoiceLatencyMetric)
	averageSendTxsLatency := getAverage(metrics, SendTxsLatencyMetric)
	averageGetPayloadLatency := getAverage(metrics, GetPayloadLatencyMetric)
	averageGasPerSecond := getAverage(metrics, GasPerSecondMetric)

	return &benchmark.SequencerKeyMetrics{
		AverageFCULatency:        averageUpdateForkChoiceLatency,
		AverageSendTxsLatency:    averageSendTxsLatency,
		AverageGetPayloadLatency: averageGetPayloadLatency,
		CommonKeyMetrics: benchmark.CommonKeyMetrics{
			AverageGasPerSecond: averageGasPerSecond,
		},
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
