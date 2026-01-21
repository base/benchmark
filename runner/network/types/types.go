package types

import (
	"crypto/ecdsa"
	"math/big"
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

	// ValidatorNodeType is the type of node used for validation. If empty, defaults to NodeType.
	ValidatorNodeType string

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

	// Include ValidatorNodeType if it's set and different from NodeType
	if p.ValidatorNodeType != "" && p.ValidatorNodeType != p.NodeType {
		params["ValidatorNodeType"] = p.ValidatorNodeType
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

const (
	UpdateForkChoiceLatencyMetric = "latency/update_fork_choice"
	NewPayloadLatencyMetric       = "latency/new_payload"
	GetPayloadLatencyMetric       = "latency/get_payload"
	SendTxsLatencyMetric          = "latency/send_txs"
	GasPerBlockMetric             = "gas/per_block"
	GasPerSecondMetric            = "gas/per_second"
	TransactionsPerBlockMetric    = "transactions/per_block"
)

type SequencerKeyMetrics struct {
	CommonKeyMetrics
	AverageFCULatency        float64 `json:"forkChoiceUpdated"`
	AverageGetPayloadLatency float64 `json:"getPayload"`
	AverageSendTxsLatency    float64 `json:"sendTxs"`
}

type ValidatorKeyMetrics struct {
	CommonKeyMetrics
	AverageNewPayloadLatency float64 `json:"newPayload"`
}

type CommonKeyMetrics struct {
	AverageGasPerSecond float64 `json:"gasPerSecond"`
}

// BlockMetricsToValidatorSummary converts block metrics to a validator summary.
func BlockMetricsToValidatorSummary(metrics []metrics.BlockMetrics) *ValidatorKeyMetrics {
	averageNewPayloadLatency := getAverage(metrics, NewPayloadLatencyMetric)
	averageGasPerSecond := getAverage(metrics, GasPerSecondMetric)

	return &ValidatorKeyMetrics{
		AverageNewPayloadLatency: averageNewPayloadLatency,
		CommonKeyMetrics: CommonKeyMetrics{
			AverageGasPerSecond: averageGasPerSecond,
		},
	}
}

// BlockMetricsToSequencerSummary converts block metrics to a sequencer summary.
func BlockMetricsToSequencerSummary(metrics []metrics.BlockMetrics) *SequencerKeyMetrics {
	averageUpdateForkChoiceLatency := getAverage(metrics, UpdateForkChoiceLatencyMetric)
	averageSendTxsLatency := getAverage(metrics, SendTxsLatencyMetric)
	averageGetPayloadLatency := getAverage(metrics, GetPayloadLatencyMetric)
	averageGasPerSecond := getAverage(metrics, GasPerSecondMetric)

	return &SequencerKeyMetrics{
		AverageFCULatency:        averageUpdateForkChoiceLatency,
		AverageSendTxsLatency:    averageSendTxsLatency,
		AverageGetPayloadLatency: averageGetPayloadLatency,
		CommonKeyMetrics: CommonKeyMetrics{
			AverageGasPerSecond: averageGasPerSecond,
		},
	}
}
