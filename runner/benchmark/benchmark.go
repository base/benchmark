package benchmark

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"sync/atomic"

	"github.com/base/base-bench/runner/network/types"
	"github.com/ethereum/go-ethereum/core"
)

// TestRun is a single run of a benchmark. Each config should result in multiple test runs.
type TestRun struct {
	ID          string
	Params      types.RunParams
	TestFile    string
	Name        string
	Description string
	OutputDir   string
}

const (
	// MaxTotalParams is the maximum number of benchmarks that can be run in parallel.
	MaxTotalParams = 100
)

var DefaultParams = &types.RunParams{
	NodeType: "geth",
	GasLimit: 50e9,
}

// NewParamsFromValues constructs a new benchmark params given a config and a set of transaction payloads to run.
func NewParamsFromValues(assignments map[string]interface{}) (*types.RunParams, error) {
	params := *DefaultParams

	for k, v := range assignments {
		if err := applyParam(&params, k, v); err != nil {
			return nil, err
		}
	}

	return &params, nil
}

func applyParam(params *types.RunParams, k string, v interface{}) error {
	if k == "params" {
		return applyParamGroup(params, v)
	}

	switch k {
	case "payload":
		if vPtrStr, ok := v.(*string); ok {
			params.PayloadID = string(*vPtrStr)
		} else if vStr, ok := v.(string); ok {
			params.PayloadID = string(vStr)
		} else {
			return fmt.Errorf("invalid payload %s", v)
		}
	case "node_type":
		if vStr, ok := v.(string); ok {
			params.NodeType = vStr
		} else {
			return fmt.Errorf("invalid node type %s", v)
		}
	case "client_bin":
		if vStr, ok := v.(string); ok {
			params.ClientBinPath = vStr
		} else {
			return fmt.Errorf("invalid client bin %s", v)
		}
	case "validator_node_type":
		if vStr, ok := v.(string); ok {
			params.ValidatorNodeType = vStr
		} else {
			return fmt.Errorf("invalid validator node type %s", v)
		}
	case "gas_limit":
		if vInt, ok := v.(int); ok {
			params.GasLimit = uint64(vInt)
		} else {
			return fmt.Errorf("invalid gas limit %s", v)
		}
	case "load_test_config":
		overrides, err := normalizeStringKeyMap(v)
		if err != nil {
			return fmt.Errorf("invalid load test config %v", v)
		}
		params.LoadTestConfigOverrides = overrides
	case "consensus_timing":
		if vStr, ok := v.(string); ok {
			if vStr != "" && vStr != types.ConsensusTimingModePreventLateFCU && vStr != types.ConsensusTimingModeBaseConsensus {
				return fmt.Errorf("invalid consensus timing %s", v)
			}
			params.ConsensusTimingMode = vStr
		} else {
			return fmt.Errorf("invalid consensus timing %s", v)
		}
	case "env":
		if vStr, ok := v.(string); ok {
			entries := strings.Split(vStr, ";")
			params.Env = make(map[string]string)
			for _, entry := range entries {
				kv := strings.Split(entry, "=")
				if len(kv) != 2 {
					return fmt.Errorf("invalid env entry %s", entry)
				}
				params.Env[kv[0]] = kv[1]
			}
		} else {
			return fmt.Errorf("invalid env %s", v)
		}
	case "num_blocks":
		if vInt, ok := v.(int); ok {
			params.NumBlocks = vInt
		} else {
			return fmt.Errorf("invalid num blocks %s", v)
		}
	case "node_args":
		// either a list of strings or a string (separated by spaces)
		if vStr, ok := v.(string); ok {
			params.NodeArgs = strings.Split(vStr, " ")
		} else if vArr, ok := v.([]interface{}); ok {
			// convert []interface{} to []string
			nodeArgs := make([]string, len(vArr))
			for i, arg := range vArr {
				arg, ok := arg.(string)
				if !ok {
					return fmt.Errorf("invalid non-string node arg %v", arg)
				}
				nodeArgs[i] = arg
			}
			params.NodeArgs = nodeArgs
		} else {
			return fmt.Errorf("invalid node args %v", v)
		}
	}
	return nil
}

func applyParamGroup(params *types.RunParams, value interface{}) error {
	// A params value groups multiple assignments into one matrix dimension,
	// preserving relationships between values that should vary together.
	expanded, err := normalizeStringKeyMap(value)
	if err != nil {
		return fmt.Errorf("invalid params %v", value)
	}
	for param, value := range expanded {
		if err := applyParam(params, param, value); err != nil {
			return fmt.Errorf("invalid params.%s: %w", param, err)
		}
	}
	return nil
}

func normalizeStringKeyMap(value interface{}) (map[string]interface{}, error) {
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed, nil
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(typed))
		for key, value := range typed {
			keyString, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("non-string key %v", key)
			}
			out[keyString] = value
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected mapping")
	}
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
