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

	// LoadTestOutputPath is the optional normal load-test report JSON path used
	// by the load-test payload worker.
	LoadTestOutputPath string
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

	// ConsensusTimingMode controls how the fake consensus client schedules FCU/getPayload calls.
	ConsensusTimingMode string

	// Env is the environment variables for the benchmark run.
	Env map[string]string

	// NumBlocks is the number of blocks to run in the benchmark run.
	NumBlocks int

	// Tags are the tags for the benchmark run.
	Tags map[string]string

	// NodeArgs are the arguments to be passed to the node binary.
	NodeArgs []string

	// ClientBinPath is an optional override for the client binary path.
	ClientBinPath string
}

const (
	ConsensusTimingModePreventLateFCU = "prevent-late-fcu"
	ConsensusTimingModeBaseConsensus  = "base-consensus"
)

func (p RunParams) UseBaseConsensusTiming() bool {
	return p.ConsensusTimingMode == ConsensusTimingModeBaseConsensus
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

	if p.ConsensusTimingMode != "" {
		params["ConsensusTimingMode"] = p.ConsensusTimingMode
	}

	for k, v := range p.Tags {
		params[k] = v
	}

	return params
}

// ClientOptions applies any client customization options to the given client options.
func (p RunParams) ClientOptions(prevClientOptions config.ClientOptions) config.ClientOptions {
	prevClientOptions.NodeArgs = p.NodeArgs
	if p.ClientBinPath != "" {
		switch p.NodeType {
		case "reth":
			prevClientOptions.RethBin = p.ClientBinPath
		case "geth":
			prevClientOptions.GethBin = p.ClientBinPath
		case "builder":
			prevClientOptions.BuilderBin = p.ClientBinPath
		case "base-reth-node":
			prevClientOptions.BaseRethNodeBin = p.ClientBinPath
		}
	}
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

// saturationGasFraction is the fraction of the MEDIAN observed gas/block a block must
// reach to count as a full (saturated) block. Saturation is measured relative to the
// median of the observed distribution rather than the configured gas limit because a
// builder under real mempool and timing constraints rarely fills a block to its nominal
// cap (observed peaks are commonly ~70% of the limit), so a limit-relative threshold
// would reject every block. Median-relative (not peak-relative) is used so a single
// exceptionally large block cannot shrink the qualifying window; blocks below this
// fraction of the median are warmup, drain, or demand-starved and would dilute the rate.
const saturationGasFraction = 0.5

const (
	RegimeCadenceLimited = "cadence-limited"
	RegimeOverloaded     = "overloaded"
	RegimeUnsaturated    = "unsaturated"
)

// median returns the median of values. It sorts a copy so the caller's slice is
// untouched. Returns 0 for an empty input.
func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// saturatedThroughput computes a duration-weighted gas/second over the saturated-block
// window and classifies the load regime.
//
// The headline rate is Σgas / Σblock_building_duration over blocks that reached
// saturationGasFraction of the median observed gas/block. This aggregate rate is invariant
// to how work is partitioned across blocks, unlike a mean of per-block rates which would
// weight a 2s block and an 11s block equally.
//
// The regime keys off get_payload latency (the actual build compute), NOT
// block_building_duration: the latter includes an intentional sleep to the block
// deadline when the builder finishes early, so it is ~blockTime even when the builder is
// idle most of the slot. A builder whose mean get_payload over the saturated window
// exceeds the whole block time cannot sustain the cadence and is overloaded; otherwise it
// finishes within the slot and is cadence-limited.
func saturatedThroughput(blockMetrics []metrics.BlockMetrics, blockTime time.Duration) (gasPerSecond float64, saturatedBlocks int, regime string) {
	var gasValues []float64
	for _, m := range blockMetrics {
		if gas, ok := m.GetMetricFloat(GasPerBlockMetric); ok {
			gasValues = append(gasValues, gas)
		}
	}
	gasThreshold := median(gasValues) * saturationGasFraction

	var sumGas, sumDuration, sumGetPayload float64
	for _, m := range blockMetrics {
		gas, hasGas := m.GetMetricFloat(GasPerBlockMetric)
		dur, hasDur := m.GetMetricFloat(BlockBuildingDurationMetric)
		if !hasGas || !hasDur || dur <= 0 {
			continue
		}
		if gasThreshold > 0 && gas < gasThreshold {
			continue
		}
		sumGas += gas
		sumDuration += dur
		if getPayload, ok := m.GetMetricFloat(GetPayloadLatencyMetric); ok {
			sumGetPayload += getPayload
		}
		saturatedBlocks++
	}

	if saturatedBlocks == 0 || sumDuration <= 0 {
		return 0, 0, RegimeUnsaturated
	}

	gasPerSecond = sumGas / sumDuration
	avgGetPayload := sumGetPayload / float64(saturatedBlocks)
	regime = RegimeCadenceLimited
	if blockTime > 0 && avgGetPayload > blockTime.Seconds() {
		regime = RegimeOverloaded
	}
	return gasPerSecond, saturatedBlocks, regime
}

const (
	UpdateForkChoiceLatencyMetric      = "latency/update_fork_choice"
	NewPayloadLatencyMetric            = "latency/new_payload"
	GetPayloadLatencyMetric            = "latency/get_payload"
	SendTxsLatencyMetric               = "latency/send_txs"
	GasPerBlockMetric                  = "gas/per_block"
	GasPerSecondMetric                 = "gas/per_second"
	BlockBuildingDurationMetric        = "duration/block_building"
	TransactionsPerBlockMetric         = "transactions/per_block"
	FlashblockProcessingDurationMetric = "reth_flashblocks_block_processing_duration"
	FlashblockSenderRecoveryMetric     = "reth_flashblocks_sender_recovery_duration"
	FlashblocksInBlockMetric           = "reth_flashblocks_flashblocks_in_block"
	FlashblockUpstreamMessagesMetric   = "reth_flashblocks_upstream_messages"
	FlashblockBundleStateCloneDuration = "reth_flashblocks_bundle_state_clone_duration"
)

type SequencerKeyMetrics struct {
	CommonKeyMetrics
	AverageFCULatency        float64 `json:"forkChoiceUpdated"`
	AverageGetPayloadLatency float64 `json:"getPayload"`
	AverageSendTxsLatency    float64 `json:"sendTxs"`
	SaturatedBlockCount      int     `json:"saturatedBlockCount"`
	Regime                   string  `json:"regime"`
}

type ValidatorKeyMetrics struct {
	CommonKeyMetrics
	AverageNewPayloadLatency            float64 `json:"newPayload"`
	AverageFlashblockProcessingDuration float64 `json:"flashblockProcessingDuration,omitempty"`
	AverageFlashblocksInBlock           float64 `json:"flashblocksInBlock,omitempty"`
}

type CommonKeyMetrics struct {
	AverageGasPerSecond float64 `json:"gasPerSecond"`
}

// BlockMetricsToValidatorSummary converts block metrics to a validator summary.
func BlockMetricsToValidatorSummary(metrics []metrics.BlockMetrics) *ValidatorKeyMetrics {
	averageNewPayloadLatency := getAverage(metrics, NewPayloadLatencyMetric)
	averageGasPerSecond := getAverage(metrics, GasPerSecondMetric)
	averageFlashblockProcessingDuration := getAverage(metrics, FlashblockProcessingDurationMetric)
	averageFlashblocksInBlock := getAverage(metrics, FlashblocksInBlockMetric)

	return &ValidatorKeyMetrics{
		AverageNewPayloadLatency:            averageNewPayloadLatency,
		AverageFlashblockProcessingDuration: averageFlashblockProcessingDuration,
		AverageFlashblocksInBlock:           averageFlashblocksInBlock,
		CommonKeyMetrics: CommonKeyMetrics{
			AverageGasPerSecond: averageGasPerSecond,
		},
	}
}

// BlockMetricsToSequencerSummary converts block metrics to a sequencer summary.
//
// AverageGasPerSecond is the duration-weighted throughput over the saturated
// (full-block) window rather than a naive mean over every block. blockTime comes from
// the run params and drives the cadence-limited/overloaded regime classification.
func BlockMetricsToSequencerSummary(metrics []metrics.BlockMetrics, blockTime time.Duration) *SequencerKeyMetrics {
	averageUpdateForkChoiceLatency := getAverage(metrics, UpdateForkChoiceLatencyMetric)
	averageSendTxsLatency := getAverage(metrics, SendTxsLatencyMetric)
	averageGetPayloadLatency := getAverage(metrics, GetPayloadLatencyMetric)
	saturatedGasPerSecond, saturatedBlocks, regime := saturatedThroughput(metrics, blockTime)

	return &SequencerKeyMetrics{
		AverageFCULatency:        averageUpdateForkChoiceLatency,
		AverageSendTxsLatency:    averageSendTxsLatency,
		AverageGetPayloadLatency: averageGetPayloadLatency,
		SaturatedBlockCount:      saturatedBlocks,
		Regime:                   regime,
		CommonKeyMetrics: CommonKeyMetrics{
			AverageGasPerSecond: saturatedGasPerSecond,
		},
	}
}
