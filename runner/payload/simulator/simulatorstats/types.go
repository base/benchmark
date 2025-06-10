package simulatorstats

import (
	"fmt"
	"maps"
	"math"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type OpcodeStats map[string]float64

func (o OpcodeStats) Round() OpcodeStats {
	result := make(OpcodeStats)
	for opcode, count := range o {
		result[opcode] = math.Round(count)
	}
	return result
}

func (o OpcodeStats) Add(other OpcodeStats) OpcodeStats {
	result := make(OpcodeStats)
	for opcode, count := range other {
		result[opcode] = o[opcode] + count
	}
	return result
}

func (o OpcodeStats) Pow(n float64) OpcodeStats {
	result := make(OpcodeStats)
	for opcode, count := range o {
		result[opcode] = math.Pow(count, n)
	}
	return result
}

func (o OpcodeStats) Sub(other OpcodeStats) OpcodeStats {
	result := make(OpcodeStats)
	for opcode, count := range other {
		result[opcode] = o[opcode] - count
	}
	return result
}

func (o OpcodeStats) Mul(n float64) OpcodeStats {
	result := make(OpcodeStats)
	for opcode, count := range o {
		result[opcode] = count * n
	}
	return result
}

func (o OpcodeStats) String() string {
	var result strings.Builder
	opcodes := make([]string, 0, len(o))
	for opcode := range o {
		opcodes = append(opcodes, opcode)
	}
	sort.Slice(opcodes, func(i, j int) bool {
		return o[opcodes[i]] > o[opcodes[j]]
	})
	opcodes = opcodes[:min(10, len(opcodes))]
	for _, opcode := range opcodes {
		result.WriteString(fmt.Sprintf("\n   - %20s: %.2f", opcode, o[opcode]))
	}
	return result.String()
}

var PrecompileAddressToName = map[common.Address]string{
	common.BytesToAddress([]byte{1}):          "ecrecover",
	common.BytesToAddress([]byte{2}):          "sha256hash",
	common.BytesToAddress([]byte{3}):          "ripemd160hash",
	common.BytesToAddress([]byte{4}):          "dataCopy",
	common.BytesToAddress([]byte{5}):          "bigModExp",
	common.BytesToAddress([]byte{6}):          "bn256Add",
	common.BytesToAddress([]byte{7}):          "bn256ScalarMul",
	common.BytesToAddress([]byte{8}):          "bn256Pairing",
	common.BytesToAddress([]byte{9}):          "blake2F",
	common.BytesToAddress([]byte{0x0a}):       "kzgPointEvaluation",
	common.BytesToAddress([]byte{0x0b}):       "bls12381G1Add",
	common.BytesToAddress([]byte{0x0c}):       "bls12381G1MultiExp",
	common.BytesToAddress([]byte{0x0d}):       "bls12381G2Add",
	common.BytesToAddress([]byte{0x0e}):       "bls12381G2MultiExp",
	common.BytesToAddress([]byte{0x0f}):       "bls12381Pairing",
	common.BytesToAddress([]byte{0x10}):       "bls12381MapG1",
	common.BytesToAddress([]byte{0x11}):       "bls12381MapG2",
	common.BytesToAddress([]byte{0x01, 0x00}): "p256Verify",
}

func (o OpcodeStats) RemoveAllBut(opcodes ...string) OpcodeStats {
	result := make(OpcodeStats)
	for _, opcode := range opcodes {
		result[opcode] = o[opcode]
	}
	return result
}

func (o OpcodeStats) Copy() OpcodeStats {
	result := make(OpcodeStats)
	maps.Copy(result, o)
	return result
}

type Stats struct {
	AccountLoaded      float64     `yaml:"account_loaded"`
	AccountDeleted     float64     `yaml:"account_deleted"`
	AccountsUpdated    float64     `yaml:"accounts_updated"`
	AccountsCreated    float64     `yaml:"accounts_created"`
	StorageLoaded      float64     `yaml:"storage_loaded"`
	StorageDeleted     float64     `yaml:"storage_deleted"`
	StorageUpdated     float64     `yaml:"storage_updated"`
	StorageCreated     float64     `yaml:"storage_created"`
	CodeSizeLoaded     float64     `yaml:"code_size_loaded"`
	NumContractsLoaded float64     `yaml:"num_contracts_loaded"`
	Opcodes            OpcodeStats `yaml:"opcodes"`
	Precompiles        OpcodeStats `yaml:"precompiles"`
}

func NewStats() *Stats {
	return &Stats{
		Opcodes:     make(OpcodeStats),
		Precompiles: make(OpcodeStats),
	}
}

func (s *Stats) Sub(other *Stats) *Stats {
	return &Stats{
		AccountLoaded:      s.AccountLoaded - other.AccountLoaded,
		AccountDeleted:     s.AccountDeleted - other.AccountDeleted,
		AccountsUpdated:    s.AccountsUpdated - other.AccountsUpdated,
		AccountsCreated:    s.AccountsCreated - other.AccountsCreated,
		StorageLoaded:      s.StorageLoaded - other.StorageLoaded,
		StorageDeleted:     s.StorageDeleted - other.StorageDeleted,
		StorageUpdated:     s.StorageUpdated - other.StorageUpdated,
		StorageCreated:     s.StorageCreated - other.StorageCreated,
		Opcodes:            s.Opcodes.Sub(other.Opcodes),
		CodeSizeLoaded:     s.CodeSizeLoaded - other.CodeSizeLoaded,
		NumContractsLoaded: s.NumContractsLoaded - other.NumContractsLoaded,
		Precompiles:        s.Precompiles.Sub(other.Precompiles),
	}
}

func (s *Stats) Pow(n float64) *Stats {
	return &Stats{
		AccountLoaded:      math.Pow(s.AccountLoaded, n),
		AccountDeleted:     math.Pow(s.AccountDeleted, n),
		AccountsUpdated:    math.Pow(s.AccountsUpdated, n),
		AccountsCreated:    math.Pow(s.AccountsCreated, n),
		StorageLoaded:      math.Pow(s.StorageLoaded, n),
		StorageDeleted:     math.Pow(s.StorageDeleted, n),
		StorageUpdated:     math.Pow(s.StorageUpdated, n),
		StorageCreated:     math.Pow(s.StorageCreated, n),
		Opcodes:            s.Opcodes.Pow(n),
		CodeSizeLoaded:     math.Pow(s.CodeSizeLoaded, n),
		NumContractsLoaded: math.Pow(s.NumContractsLoaded, n),
		Precompiles:        s.Precompiles.Pow(n),
	}
}

func (s *Stats) Add(other *Stats) *Stats {
	return &Stats{
		AccountLoaded:      s.AccountLoaded + other.AccountLoaded,
		AccountDeleted:     s.AccountDeleted + other.AccountDeleted,
		AccountsUpdated:    s.AccountsUpdated + other.AccountsUpdated,
		AccountsCreated:    s.AccountsCreated + other.AccountsCreated,
		StorageLoaded:      s.StorageLoaded + other.StorageLoaded,
		StorageDeleted:     s.StorageDeleted + other.StorageDeleted,
		StorageUpdated:     s.StorageUpdated + other.StorageUpdated,
		StorageCreated:     s.StorageCreated + other.StorageCreated,
		Opcodes:            s.Opcodes.Add(other.Opcodes),
		CodeSizeLoaded:     s.CodeSizeLoaded + other.CodeSizeLoaded,
		NumContractsLoaded: s.NumContractsLoaded + other.NumContractsLoaded,
		Precompiles:        s.Precompiles.Add(other.Precompiles),
	}
}

func (s *Stats) Mul(n float64) *Stats {
	return &Stats{
		AccountLoaded:      s.AccountLoaded * n,
		AccountDeleted:     s.AccountDeleted * n,
		AccountsUpdated:    s.AccountsUpdated * n,
		AccountsCreated:    s.AccountsCreated * n,
		StorageLoaded:      s.StorageLoaded * n,
		StorageDeleted:     s.StorageDeleted * n,
		StorageUpdated:     s.StorageUpdated * n,
		StorageCreated:     s.StorageCreated * n,
		Opcodes:            s.Opcodes.Mul(n),
		CodeSizeLoaded:     s.CodeSizeLoaded * n,
		NumContractsLoaded: s.NumContractsLoaded * n,
		Precompiles:        s.Precompiles.Mul(n),
	}
}

func (s *Stats) Round() *Stats {
	return &Stats{
		AccountLoaded:      math.Round(s.AccountLoaded),
		AccountDeleted:     math.Round(s.AccountDeleted),
		AccountsUpdated:    math.Round(s.AccountsUpdated),
		AccountsCreated:    math.Round(s.AccountsCreated),
		StorageLoaded:      math.Round(s.StorageLoaded),
		StorageDeleted:     math.Round(s.StorageDeleted),
		StorageUpdated:     math.Round(s.StorageUpdated),
		StorageCreated:     math.Round(s.StorageCreated),
		Opcodes:            s.Opcodes.Round(),
		CodeSizeLoaded:     math.Round(s.CodeSizeLoaded),
		NumContractsLoaded: math.Round(s.NumContractsLoaded),
		Precompiles:        s.Precompiles.Round(),
	}
}

func (s *Stats) Copy() *Stats {
	return &Stats{
		AccountLoaded:      s.AccountLoaded,
		AccountDeleted:     s.AccountDeleted,
		AccountsUpdated:    s.AccountsUpdated,
		AccountsCreated:    s.AccountsCreated,
		StorageLoaded:      s.StorageLoaded,
		StorageDeleted:     s.StorageDeleted,
		StorageUpdated:     s.StorageUpdated,
		StorageCreated:     s.StorageCreated,
		CodeSizeLoaded:     s.CodeSizeLoaded,
		NumContractsLoaded: s.NumContractsLoaded,
		Opcodes:            s.Opcodes.Copy(),
		Precompiles:        s.Precompiles.Copy(),
	}
}

func (s *Stats) String() string {
	res := fmt.Sprintf("- Accounts Reads: %.2f\n", s.AccountLoaded)
	res += fmt.Sprintf("- Accounts Deletes: %.2f\n", s.AccountDeleted)
	res += fmt.Sprintf("- Accounts Updates: %.2f\n", s.AccountsUpdated)
	res += fmt.Sprintf("- Accounts Created: %.2f\n", s.AccountsCreated)
	res += fmt.Sprintf("- Storage Reads: %.2f\n", s.StorageLoaded)
	res += fmt.Sprintf("- Storage Deletes: %.2f\n", s.StorageDeleted)
	res += fmt.Sprintf("- Storage Updates: %.2f\n", s.StorageUpdated)
	res += fmt.Sprintf("- Storage Created: %.2f\n", s.StorageCreated)
	res += fmt.Sprintf("- Code Size Loaded: %.2f\n", s.CodeSizeLoaded)
	res += fmt.Sprintf("- Number of Contracts Loaded: %.2f\n", s.NumContractsLoaded)
	res += fmt.Sprintf("- Opcode Stats: %s\n", s.Opcodes.String())
	res += fmt.Sprintf("- Precompile Stats: %s\n", s.Precompiles.String())
	return res
}
