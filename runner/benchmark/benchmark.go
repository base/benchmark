package benchmark

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/base/base-bench/runner/config"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type TransactionPayload string

// Params is the parameters for a single benchmark run.
type Params struct {
	NodeType           string
	GasLimit           uint64
	TransactionPayload TransactionPayload
	BlockTime          time.Duration
	Env                map[string]string
	NumBlocks          int
	Tags               map[string]string
}

func (p Params) ToConfig() map[string]interface{} {
	params := map[string]interface{}{
		"NodeType":           p.NodeType,
		"GasLimit":           p.GasLimit,
		"TransactionPayload": p.TransactionPayload,
	}

	for k, v := range p.Tags {
		params[k] = v
	}

	return params
}

// TestRun is a single run of a benchmark. Each config should result in multiple test runs.
type TestRun struct {
	ID          string
	Params      Params
	TestFile    string
	Name        string
	Description string
	OutputDir   string
}

const (
	// MaxTotalParams is the maximum number of benchmarks that can be run in parallel.
	MaxTotalParams = 24
)

var DefaultParams = &Params{
	NodeType:  "geth",
	GasLimit:  50e9,
	BlockTime: 1 * time.Second,
}

// NewParamsFromValues constructs a new benchmark params given a config and a set of transaction payloads to run.
func NewParamsFromValues(assignments map[ParamType]interface{}) (*Params, error) {
	params := *DefaultParams

	for k, v := range assignments {
		switch k {
		case ParamTypeTxWorkload:
			if vPtrStr, ok := v.(*string); ok {
				params.TransactionPayload = TransactionPayload(*vPtrStr)
			} else if vStr, ok := v.(string); ok {
				params.TransactionPayload = TransactionPayload(vStr)
			} else {
				return nil, fmt.Errorf("invalid transaction workload %s", v)
			}
		case ParamTypeNode:
			if vStr, ok := v.(string); ok {
				params.NodeType = vStr
			} else {
				return nil, fmt.Errorf("invalid node type %s", v)
			}
		case ParamTypeGasLimit:
			if vInt, ok := v.(int); ok {
				params.GasLimit = uint64(vInt)
			} else {
				return nil, fmt.Errorf("invalid gas limit %s", v)
			}
		case ParamTypeEnv:
			if vStr, ok := v.(string); ok {
				entries := strings.Split(vStr, ";")
				params.Env = make(map[string]string)
				for _, entry := range entries {
					kv := strings.Split(entry, "=")
					if len(kv) != 2 {
						return nil, fmt.Errorf("invalid env entry %s", entry)
					}
					params.Env[kv[0]] = kv[1]
				}
			} else {
				return nil, fmt.Errorf("invalid env %s", v)
			}
		case ParamTypeNumBlocks:
			if vInt, ok := v.(int); ok {
				params.NumBlocks = vInt
			} else {
				return nil, fmt.Errorf("invalid num blocks %s", v)
			}
		}
	}

	return &params, nil
}

// ClientOptions applies any client customization options to the given client options.
func (p Params) ClientOptions(prevClientOptions config.ClientOptions) config.ClientOptions {
	return prevClientOptions
}

const MAX_GAS_LIMIT = math.MaxUint64

// DefaultGenesis returns the genesis block for a devnet.
func DefaultDevnetGenesis(genesisTime time.Time) *core.Genesis {
	zero := uint64(0)
	fifty := uint64(50)

	allocs := make(gethTypes.GenesisAlloc)
	return &core.Genesis{
		Nonce:      0,
		Timestamp:  uint64(genesisTime.Unix()),
		ExtraData:  eip1559.EncodeHoloceneExtraData(50, 1),
		GasLimit:   MAX_GAS_LIMIT, // adjust gas limit in FCU call, not here
		Difficulty: big.NewInt(1),
		Alloc:      allocs,
		Config: &params.ChainConfig{
			ChainID: big.NewInt(13371337),
			// Ethereum forks in proof-of-work era.
			HomesteadBlock:      big.NewInt(0),
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			MuirGlacierBlock:    big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			ArrowGlacierBlock:   big.NewInt(0),
			GrayGlacierBlock:    big.NewInt(0),
			MergeNetsplitBlock:  big.NewInt(0),
			// Ethereum forks in proof-of-stake era.
			TerminalTotalDifficulty: big.NewInt(1),
			ShanghaiTime:            new(uint64),
			CancunTime:              new(uint64),
			PragueTime:              new(uint64),
			VerkleTime:              nil,
			// OP-Stack forks are disabled, since we use this for L1.
			BedrockBlock: big.NewInt(0),
			RegolithTime: &zero,
			CanyonTime:   &zero,
			EcotoneTime:  &zero,
			FjordTime:    &zero,
			GraniteTime:  &zero,
			HoloceneTime: &zero,
			IsthmusTime:  &zero,
			InteropTime:  nil,
			Optimism: &params.OptimismConfig{
				EIP1559Elasticity:        1,
				EIP1559Denominator:       50,
				EIP1559DenominatorCanyon: &fifty,
			},
		},
	}
}

type Benchmark struct {
	Params Params
}

func NewBenchmark() *Benchmark {
	return &Benchmark{}
}
