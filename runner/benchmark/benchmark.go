package benchmark

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/base/base-bench/runner/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type TransactionPayload struct {
	Type string
}

// Params is the parameters for a single benchmark run.
type Params struct {
	NodeType           string
	TransactionPayload []TransactionPayload
	BlockTime          time.Duration
	Env                map[string]string
}

// ParamsMatrix is a list of params that can be run in parallel.
type ParamsMatrix []Params

const (
	// MaxTotalParams is the maximum number of benchmarks that can be run in parallel.
	MaxTotalParams = 24
)

// NewParamsFromValues constructs a new benchmark params given a config and a set of transaction payloads to run.
func NewParamsFromValues(assignments map[ParamType]string, transactionPayloads []TransactionPayload) (*Params, error) {
	params := Params{
		NodeType:           "geth",
		TransactionPayload: transactionPayloads,
		BlockTime:          1 * time.Second,
	}

	for k, v := range assignments {
		switch k {
		case ParamTypeNode:
			params.NodeType = v
		case ParamTypeEnv:
			entries := strings.Split(v, ";")
			params.Env = make(map[string]string)
			for _, entry := range entries {
				kv := strings.Split(entry, "=")
				if len(kv) != 2 {
					return nil, fmt.Errorf("invalid env entry %s", entry)
				}
				params.Env[kv[0]] = kv[1]
			}
		}
	}

	return &params, nil
}

// ClientOptions applies any client customization options to the given client options.
func (p Params) ClientOptions(prevClientOptions config.ClientOptions) config.ClientOptions {
	return prevClientOptions
}

// Genesis returns the genesis block for a given genesis time.
func (p Params) Genesis(genesisTime time.Time) core.Genesis {
	zero := uint64(0)
	fifty := uint64(50)

	allocs := make(gethTypes.GenesisAlloc)
	// private key: 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
	allocs[common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")] = gethTypes.Account{
		Balance: new(big.Int).Mul(big.NewInt(1e6), big.NewInt(params.Ether)), // 100,000 ETH
	}

	return core.Genesis{
		Nonce:      0,
		Timestamp:  uint64(genesisTime.Unix()),
		ExtraData:  eip1559.EncodeHoloceneExtraData(50, 10),
		GasLimit:   40e9,
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
			InteropTime:  &zero,
			Optimism: &params.OptimismConfig{
				EIP1559Elasticity:        10,
				EIP1559Denominator:       50,
				EIP1559DenominatorCanyon: &fifty,
			},
		},
	}
}

func parseTransactionPayloads(payloads []string) ([]TransactionPayload, error) {
	var txPayloads []TransactionPayload
	for _, p := range payloads {
		txPayloads = append(txPayloads, TransactionPayload{Type: p})
	}
	return txPayloads, nil
}

// NewParamsMatrixFromConfig constructs a new ParamsMatrix from a config.
func NewParamsMatrixFromConfig(c Matrix) (ParamsMatrix, error) {
	var txPayloadOptions []TransactionPayload

	seenParams := make(map[ParamType]bool)

	// Multiple payloads can run in a single benchmark.
	paramsExceptPayload := make([]Param, 0, len(c.Variables))
	for _, p := range c.Variables {
		if seenParams[p.ParamType] {
			return nil, fmt.Errorf("duplicate param type %s", p.ParamType)
		}
		seenParams[p.ParamType] = true
		if p.ParamType == ParamTypeTxWorkload {
			var params []string
			if p.Values != nil {
				params = *p.Values
			} else {
				params = []string{*p.Value}
			}
			options, err := parseTransactionPayloads(params)
			if err != nil {
				return nil, err
			}
			txPayloadOptions = options
			continue
		}
		paramsExceptPayload = append(paramsExceptPayload, p)
	}

	// Ensure transaction payload is specified
	if txPayloadOptions == nil {
		return nil, fmt.Errorf("no transaction payloads specified")
	}

	// Calculate the dimensions of the matrix for each param
	dimensions := make([]int, len(paramsExceptPayload))
	for i, p := range paramsExceptPayload {
		if p.Values != nil {
			dimensions[i] = len(*p.Values)
		} else {
			dimensions[i] = 1
		}
	}

	// Create a list of values for each param
	valuesByParam := make([][]string, len(paramsExceptPayload))
	for i, p := range paramsExceptPayload {
		if p.Values == nil {
			valuesByParam[i] = []string{*p.Value}
		} else {
			valuesByParam[i] = *p.Values
		}
	}

	// Ensure total params is less than the max
	totalParams := 1
	for _, d := range dimensions {
		totalParams *= d
	}

	if totalParams > MaxTotalParams {
		return nil, fmt.Errorf("total number of params %d exceeds max %d", totalParams, MaxTotalParams)
	}

	currentParams := make([]int, len(dimensions))

	// Create the params matrix
	paramsMatrix := make(ParamsMatrix, totalParams)

	for i := 0; i < totalParams; i++ {
		valueSelections := make(map[ParamType]string)
		for j, p := range paramsExceptPayload {
			valueSelections[p.ParamType] = valuesByParam[j][currentParams[j]]
		}

		params, err := NewParamsFromValues(valueSelections, txPayloadOptions)
		if err != nil {
			return nil, err
		}
		paramsMatrix[i] = *params

		done := true

		// Increment current params from the rightmost param
		for incIdx := len(dimensions) - 1; incIdx >= 0; incIdx-- {
			// find the next param that is incrementable
			if currentParams[incIdx] < dimensions[incIdx]-1 {
				currentParams[incIdx]++
				done = false
				break
			} else {
				// If this param is currently at the max, reset it to 0 and continue to the next param
				currentParams[incIdx] = 0
			}
		}

		if done {
			break
		}
	}

	return paramsMatrix, nil
}

type Benchmark struct {
	Params Params
}

func NewBenchmark() *Benchmark {
	return &Benchmark{}
}
