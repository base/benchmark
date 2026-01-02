package types

import (
	"crypto/ecdsa"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/base/base-bench/runner/config"
	"github.com/base/base-bench/runner/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// BasicBlockType implements what chain config would usually implement.
type IsthmusBlockType struct{}

// HasOptimismWithdrawalsRoot implements types.BlockType.
func (b IsthmusBlockType) HasOptimismWithdrawalsRoot(blkTime uint64) bool {
	return true
}

// IsIsthmus implements types.BlockType.
func (b IsthmusBlockType) IsIsthmus(blkTime uint64) bool {
	return true
}

var _ ethTypes.BlockType = IsthmusBlockType{}

// TestConfig holds all configuration needed for a benchmark test
type TestConfig struct {
	Params     RunParams
	Config     config.Config
	Genesis    core.Genesis
	BatcherKey ecdsa.PrivateKey
	// BatcherAddr is lazily initialized to avoid unnecessary computation
	batcherAddr *common.Address

	PrefundPrivateKey ecdsa.PrivateKey
	PrefundAmount     big.Int
}

// BatcherAddr returns the batcher address, computing it if necessary
func (c *TestConfig) BatcherAddr() common.Address {
	if c.batcherAddr == nil {
		batcherAddr := crypto.PubkeyToAddress(c.BatcherKey.PublicKey)
		c.batcherAddr = &batcherAddr
	}
	return *c.batcherAddr
}

// Params is the parameters for a single benchmark run.
type RunParams struct {
	// NodeType is the type of node that's being benchmarked. Examples: geth, reth, nethermined, etc.
	NodeType string

	// GasLimit is the gas limit for the benchmark run which is the maximum gas that the sequencer will include per block.
	GasLimit uint64

	// PayloadID is a reference to a transaction payload that will be sent to the sequencer.
	PayloadID string

	// BenchmarkRunID is a unique identifier for the benchmark run.
	BenchmarkRunID string

	// Name is the name of the benchmark run in the config file.
	Name string

	// Description is the description of the benchmark run in the config file.
	Description string

	// BlockTime is the time between blocks in the benchmark run.
	BlockTime time.Duration

	// Env is the environment variables for the benchmark run.
	Env map[string]string

	// NumBlocks is the number of blocks to run in the benchmark run.
	NumBlocks int

	// Tags are the tags for the benchmark run.
	Tags map[string]string

	// NodeArgs are the arguments to be passed to the node binary.
	NodeArgs []string
}

func (p RunParams) ToConfig() map[string]interface{} {
	params := map[string]interface{}{
		"NodeType":              p.NodeType,
		"GasLimit":              p.GasLimit,
		"TransactionPayload":    p.PayloadID,
		"BenchmarkRun":          p.BenchmarkRunID,
		"BlockTimeMilliseconds": p.BlockTime.Milliseconds(),
		"NodeArgs":              strings.Join(p.NodeArgs, " "),
	}

	for k, v := range p.Tags {
		params[k] = v
	}

	return params
}

// ClientOptions applies any client customization options to the given client options.
func (p RunParams) ClientOptions(prevClientOptions config.ClientOptions) config.ClientOptions {
	prevClientOptions.NodeArgs = p.NodeArgs
	return prevClientOptions
}

func getAverage(metrics []metrics.BlockMetrics, metricName string) float64 {
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

func getPercentile(metrics []metrics.BlockMetrics, metricName string, percentile float64) float64 {
	var values []float64
	for _, metric := range metrics {
		if value, ok := metric.GetMetricFloat(metricName); ok {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	index := int(float64(len(values)-1) * percentile / 100)
	return values[index]
}

// LatencyStats holds average and percentile statistics for a latency metric.
type LatencyStats struct {
	Avg float64 `json:"avg"`
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
	P99 float64 `json:"p99"`
}

func getLatencyStats(metrics []metrics.BlockMetrics, metricName string) LatencyStats {
	return LatencyStats{
		Avg: getAverage(metrics, metricName),
		P50: getPercentile(metrics, metricName, 50),
		P90: getPercentile(metrics, metricName, 90),
		P99: getPercentile(metrics, metricName, 99),
	}
}

const (
	UpdateForkChoiceLatencyMetric    = "latency/update_fork_choice"
	NewPayloadLatencyMetric          = "latency/new_payload"
	SequencerNewPayloadLatencyMetric = "latency/sequencer_new_payload"
	GetPayloadLatencyMetric          = "latency/get_payload"
	SendTxsLatencyMetric             = "latency/send_txs"
	GasPerBlockMetric                = "gas/per_block"
	GasPerSecondMetric               = "gas/per_second"
	TransactionsPerBlockMetric       = "transactions/per_block"
)

type SequencerKeyMetrics struct {
	CommonKeyMetrics
	FCULatency        LatencyStats `json:"forkChoiceUpdated"`
	GetPayloadLatency LatencyStats `json:"getPayload"`
	NewPayloadLatency LatencyStats `json:"newPayload"`
	SendTxsLatency    LatencyStats `json:"sendTxs"`
}

type ValidatorKeyMetrics struct {
	CommonKeyMetrics
	NewPayloadLatency LatencyStats `json:"newPayload"`
}

type CommonKeyMetrics struct {
	AverageGasPerSecond float64 `json:"gasPerSecond"`
}

// BlockMetricsToValidatorSummary converts block metrics to a validator summary.
func BlockMetricsToValidatorSummary(metrics []metrics.BlockMetrics) *ValidatorKeyMetrics {
	newPayloadLatency := getLatencyStats(metrics, NewPayloadLatencyMetric)
	averageGasPerSecond := getAverage(metrics, GasPerSecondMetric)

	return &ValidatorKeyMetrics{
		NewPayloadLatency: newPayloadLatency,
		CommonKeyMetrics: CommonKeyMetrics{
			AverageGasPerSecond: averageGasPerSecond,
		},
	}
}

// BlockMetricsToSequencerSummary converts block metrics to a sequencer summary.
func BlockMetricsToSequencerSummary(metrics []metrics.BlockMetrics) *SequencerKeyMetrics {
	fcuLatency := getLatencyStats(metrics, UpdateForkChoiceLatencyMetric)
	sendTxsLatency := getLatencyStats(metrics, SendTxsLatencyMetric)
	getPayloadLatency := getLatencyStats(metrics, GetPayloadLatencyMetric)
	newPayloadLatency := getLatencyStats(metrics, SequencerNewPayloadLatencyMetric)
	averageGasPerSecond := getAverage(metrics, GasPerSecondMetric)

	return &SequencerKeyMetrics{
		FCULatency:        fcuLatency,
		SendTxsLatency:    sendTxsLatency,
		GetPayloadLatency: getPayloadLatency,
		NewPayloadLatency: newPayloadLatency,
		CommonKeyMetrics: CommonKeyMetrics{
			AverageGasPerSecond: averageGasPerSecond,
		},
	}
}
