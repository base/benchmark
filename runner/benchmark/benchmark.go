package benchmark

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/base/base-bench/runner/config"
	"github.com/ethereum/go-ethereum/core"
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
func NewParamsFromValues(assignments map[string]interface{}) (*Params, error) {
	params := *DefaultParams

	for k, v := range assignments {
		switch k {
		case "transaction_workload":
			if vPtrStr, ok := v.(*string); ok {
				params.TransactionPayload = TransactionPayload(*vPtrStr)
			} else if vStr, ok := v.(string); ok {
				params.TransactionPayload = TransactionPayload(vStr)
			} else {
				return nil, fmt.Errorf("invalid transaction workload %s", v)
			}
		case "node_type":
			if vStr, ok := v.(string); ok {
				params.NodeType = vStr
			} else {
				return nil, fmt.Errorf("invalid node type %s", v)
			}
		case "gas_limit":
			if vInt, ok := v.(int); ok {
				params.GasLimit = uint64(vInt)
			} else {
				return nil, fmt.Errorf("invalid gas limit %s", v)
			}
		case "env":
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
		case "num_blocks":
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

var cachedGenesis atomic.Pointer[core.Genesis]

// DefaultGenesis returns the genesis block for a devnet.
func DefaultDevnetGenesis() *core.Genesis {
	if genesis := cachedGenesis.Load(); genesis != nil {
		return genesis
	}
	// read from genesis.json
	var genesis core.Genesis

	f, err := os.OpenFile("./genesis.json", os.O_RDONLY, 0644)

	if err != nil {
		panic(fmt.Sprintf("failed to open genesis.json: %v", err))
	}
	defer func() {
		_ = f.Close()
	}()

	if err := json.NewDecoder(f).Decode(&genesis); err != nil {
		panic(fmt.Sprintf("failed to decode genesis.json: %v", err))
	}

	cachedGenesis.CompareAndSwap(nil, &genesis)

	return &genesis
}

type Benchmark struct {
	Params Params
}

func NewBenchmark() *Benchmark {
	return &Benchmark{}
}
