package main

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"

	"maps"

	"github.com/ethereum-optimism/optimism/op-program/chainconfig"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

func fetchBlockStats(log log.Logger, client *ethclient.Client, block *types.Block, genesis *core.Genesis, headerCache map[common.Hash]*types.Header) (*stats, []*stats, error) {
	log.Info("Fetching execution witness")

	var result *eth.ExecutionWitness
	err := client.Client().CallContext(context.Background(), &result, "debug_executionWitness", hexutil.EncodeUint64(block.NumberU64()))
	if err != nil {
		return nil, nil, err
	}

	log.Info("Finished fetching execution witness")

	parentBlock, err := client.BlockByHash(context.Background(), block.ParentHash())
	if err != nil {
		return nil, nil, err
	}

	return executeBlock(log, client, parentBlock, block, result, genesis, headerCache)
}

type blockCtx struct {
	engine                consensus.Engine
	getHeaderByHashNumber func(hash common.Hash, number uint64) *types.Header
	config                *params.ChainConfig
	headers               map[common.Hash]*types.Header
}

func newBlockCtx(genesis *core.Genesis, ethClient *ethclient.Client, headerCache map[common.Hash]*types.Header) *blockCtx {
	getHeaderByHashNumber := func(hash common.Hash, number uint64) *types.Header {
		header, err := ethClient.HeaderByHash(context.Background(), hash)
		if err != nil {
			panic(err)
		}
		return header
	}

	return &blockCtx{
		engine:                beacon.New(nil),
		getHeaderByHashNumber: getHeaderByHashNumber,
		config:                genesis.Config,
		headers:               headerCache,
	}
}

func (b *blockCtx) Engine() consensus.Engine {
	return b.engine
}

func (b *blockCtx) GetHeader(hash common.Hash, number uint64) *types.Header {
	if header, ok := b.headers[hash]; ok {
		return header
	}
	header := b.getHeaderByHashNumber(hash, number)
	b.headers[hash] = header
	return header
}

func (b *blockCtx) Config() *params.ChainConfig {
	return b.config
}

type opcodeStats map[string]float64

func (o opcodeStats) add(other opcodeStats) opcodeStats {
	result := make(opcodeStats)
	for opcode, count := range other {
		result[opcode] = o[opcode] + count
	}
	return result
}

func (o opcodeStats) pow(n float64) opcodeStats {
	result := make(opcodeStats)
	for opcode, count := range o {
		result[opcode] = math.Pow(count, n)
	}
	return result
}

func (o opcodeStats) sub(other opcodeStats) opcodeStats {
	result := make(opcodeStats)
	for opcode, count := range other {
		result[opcode] = o[opcode] - count
	}
	return result
}

func (o opcodeStats) mul(n float64) opcodeStats {
	result := make(opcodeStats)
	for opcode, count := range o {
		result[opcode] = count * n
	}
	return result
}

func (o opcodeStats) String() string {
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

var allPrecompiles = map[common.Address]string{
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

func (o opcodeStats) removeAllBut(opcodes ...string) opcodeStats {
	result := make(opcodeStats)
	for _, opcode := range opcodes {
		result[opcode] = o[opcode]
	}
	return result
}

func (o opcodeStats) copy() opcodeStats {
	result := make(opcodeStats)
	maps.Copy(result, o)
	return result
}

type stats struct {
	accountLoaded      float64
	accountDeleted     float64
	accountsUpdated    float64
	storageLoaded      float64
	storageDeleted     float64
	storageUpdated     float64
	codeSizeLoaded     float64
	numContractsLoaded float64
	opcodes            opcodeStats
	precompileStats    opcodeStats
}

func newStats() *stats {
	return &stats{
		accountLoaded:      0,
		accountDeleted:     0,
		accountsUpdated:    0,
		storageLoaded:      0,
		storageDeleted:     0,
		storageUpdated:     0,
		codeSizeLoaded:     0,
		numContractsLoaded: 0,
		opcodes:            make(opcodeStats),
	}
}

func (s *stats) update(db *state.StateDB, codePrestate map[common.Hash][]byte, opcodeStats opcodeStats, precompileStats opcodeStats) {
	s.accountLoaded = float64(db.AccountLoaded)
	s.accountDeleted = float64(db.AccountDeleted)
	s.accountsUpdated = float64(db.AccountUpdated)
	s.storageLoaded = float64(db.StorageLoaded)
	s.storageDeleted = float64(db.StorageDeleted.Load())
	s.storageUpdated = float64(db.StorageUpdated.Load())

	totalCodeSize := uint64(0)
	for _, code := range codePrestate {
		totalCodeSize += uint64(len(code))
	}

	s.codeSizeLoaded = float64(totalCodeSize)
	s.numContractsLoaded = float64(len(codePrestate))
	s.opcodes = opcodeStats.removeAllBut("EXP", "KECCAK256")
	s.precompileStats = precompileStats
}

func (s *stats) sub(other *stats) *stats {
	return &stats{
		accountLoaded:      s.accountLoaded - other.accountLoaded,
		accountDeleted:     s.accountDeleted - other.accountDeleted,
		accountsUpdated:    s.accountsUpdated - other.accountsUpdated,
		storageLoaded:      s.storageLoaded - other.storageLoaded,
		storageDeleted:     s.storageDeleted - other.storageDeleted,
		storageUpdated:     s.storageUpdated - other.storageUpdated,
		opcodes:            s.opcodes.sub(other.opcodes),
		codeSizeLoaded:     s.codeSizeLoaded - other.codeSizeLoaded,
		numContractsLoaded: s.numContractsLoaded - other.numContractsLoaded,
		precompileStats:    s.precompileStats.sub(other.precompileStats),
	}
}

func (s *stats) pow(n float64) *stats {
	return &stats{
		accountLoaded:      math.Pow(s.accountLoaded, n),
		accountDeleted:     math.Pow(s.accountDeleted, n),
		accountsUpdated:    math.Pow(s.accountsUpdated, n),
		storageLoaded:      math.Pow(s.storageLoaded, n),
		storageDeleted:     math.Pow(s.storageDeleted, n),
		storageUpdated:     math.Pow(s.storageUpdated, n),
		opcodes:            s.opcodes.pow(n),
		codeSizeLoaded:     math.Pow(s.codeSizeLoaded, n),
		numContractsLoaded: math.Pow(s.numContractsLoaded, n),
		precompileStats:    s.precompileStats.pow(n),
	}
}

func (s *stats) add(other *stats) *stats {
	return &stats{
		accountLoaded:      s.accountLoaded + other.accountLoaded,
		accountDeleted:     s.accountDeleted + other.accountDeleted,
		accountsUpdated:    s.accountsUpdated + other.accountsUpdated,
		storageLoaded:      s.storageLoaded + other.storageLoaded,
		storageDeleted:     s.storageDeleted + other.storageDeleted,
		storageUpdated:     s.storageUpdated + other.storageUpdated,
		opcodes:            s.opcodes.add(other.opcodes),
		codeSizeLoaded:     s.codeSizeLoaded + other.codeSizeLoaded,
		numContractsLoaded: s.numContractsLoaded + other.numContractsLoaded,
		precompileStats:    s.precompileStats.add(other.precompileStats),
	}
}

func (s *stats) mul(n float64) *stats {
	return &stats{
		accountLoaded:      s.accountLoaded * n,
		accountDeleted:     s.accountDeleted * n,
		accountsUpdated:    s.accountsUpdated * n,
		storageLoaded:      s.storageLoaded * n,
		storageDeleted:     s.storageDeleted * n,
		storageUpdated:     s.storageUpdated * n,
		opcodes:            s.opcodes.mul(n),
		codeSizeLoaded:     s.codeSizeLoaded * n,
		numContractsLoaded: s.numContractsLoaded * n,
		precompileStats:    s.precompileStats.mul(n),
	}
}

func (s *stats) copy() *stats {
	return &stats{
		accountLoaded:      s.accountLoaded,
		accountDeleted:     s.accountDeleted,
		accountsUpdated:    s.accountsUpdated,
		storageLoaded:      s.storageLoaded,
		storageDeleted:     s.storageDeleted,
		storageUpdated:     s.storageUpdated,
		codeSizeLoaded:     s.codeSizeLoaded,
		numContractsLoaded: s.numContractsLoaded,
		opcodes:            s.opcodes.copy(),
		precompileStats:    s.precompileStats.copy(),
	}
}

func (s *stats) String() string {
	res := fmt.Sprintf("- Accounts Reads: %.2f\n", s.accountLoaded)
	res += fmt.Sprintf("- Accounts Deletes: %.2f\n", s.accountDeleted)
	res += fmt.Sprintf("- Accounts Updates: %.2f\n", s.accountsUpdated)
	res += fmt.Sprintf("- Storage Reads: %.2f\n", s.storageLoaded)
	res += fmt.Sprintf("- Storage Deletes: %.2f\n", s.storageDeleted)
	res += fmt.Sprintf("- Storage Updates: %.2f\n", s.storageUpdated)
	res += fmt.Sprintf("- Code Size Loaded: %.2f\n", s.codeSizeLoaded)
	res += fmt.Sprintf("- Number of Contracts Loaded: %.2f\n", s.numContractsLoaded)
	res += fmt.Sprintf("- Opcode Stats: %s\n", s.opcodes.String())
	res += fmt.Sprintf("- Precompile Stats: %s\n", s.precompileStats.String())
	return res
}

func executeBlock(log log.Logger, client *ethclient.Client, parent *types.Block, executedBlock *types.Block, witness *eth.ExecutionWitness, genesis *core.Genesis, headerCache map[common.Hash]*types.Header) (*stats, []*stats, error) {
	header := &types.Header{
		ParentHash:      parent.Hash(),
		Coinbase:        executedBlock.Coinbase(),
		Difficulty:      executedBlock.Difficulty(),
		Number:          executedBlock.Number(),
		GasLimit:        executedBlock.GasLimit(),
		Time:            executedBlock.Time(),
		Extra:           executedBlock.Extra(),
		MixDigest:       executedBlock.MixDigest(),
		WithdrawalsHash: executedBlock.WithdrawalsRoot(),
		RequestsHash:    executedBlock.RequestsHash(),
	}

	codes := make(map[common.Hash][]byte)
	nodes := make(map[common.Hash][]byte)

	chainCfg, err := chainconfig.ChainConfigByChainID(eth.ChainIDFromBig(big.NewInt(8453)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chain config: %w", err)
	}

	genesis.Config = chainCfg

	chainCtx := newBlockCtx(genesis, client, headerCache)

	for _, code := range witness.Codes {
		codes[crypto.Keccak256Hash(code)] = []byte(code)
	}

	for _, node := range witness.State {
		nodes[crypto.Keccak256Hash(node)] = []byte(node)
	}

	db := memorydb.New()
	oracleKv := newPreimageOracle(db, codes, nodes)
	oracleDb := NewOracleBackedDB(db, oracleKv, eth.ChainIDFromBig(genesis.Config.ChainID))

	// copied from geth:
	statedb, err := state.New(parent.Root(), state.NewDatabase(triedb.NewDatabase(rawdb.NewDatabase(oracleDb), nil), nil))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init state db around block %s (state %s): %w", parent.Hash().Hex(), parent.Root().Hex(), err)
	}

	blockStats := newStats()
	txStats := make([]*stats, len(executedBlock.Transactions()))

	if genesis.Config.IsLondon(header.Number) {
		header.BaseFee = eip1559.CalcBaseFee(genesis.Config, parent.Header(), header.Time)
		// At the transition, double the gas limit so the gas target is equal to the old gas limit.
		if !genesis.Config.IsLondon(parent.Number()) {
			header.GasLimit = parent.GasLimit() * genesis.Config.ElasticityMultiplier()
		}
	}
	blockTracer := newOpcodeTracer()

	if genesis.Config.IsCancun(header.Number, header.Time) {
		header.BlobGasUsed = new(uint64)
		excessBlobGas := eip4844.CalcExcessBlobGas(genesis.Config, parent.Header(), header.Time)
		header.ExcessBlobGas = &excessBlobGas
		root := crypto.Keccak256Hash([]byte("fake-beacon-block-root"), header.Number.Bytes())
		header.ParentBeaconRoot = &root

		context := core.NewEVMBlockContext(header, chainCtx, nil, genesis.Config, statedb)
		var precompileOverrides vm.PrecompileOverrides

		vmenv := vm.NewEVM(context, statedb, genesis.Config, vm.Config{PrecompileOverrides: precompileOverrides, Tracer: blockTracer.Tracer()})
		core.ProcessBeaconBlockRoot(*header.ParentBeaconRoot, vmenv)

		if genesis.Config.IsPrague(header.Number, header.Time) {
			core.ProcessParentBlockHash(header.ParentHash, vmenv)
		}
	}

	gasPool := new(core.GasPool)
	gasPool.AddGas(header.GasLimit)

	blockStats.update(statedb, codes, blockTracer.opcodeStats, blockTracer.precompileStats)

	log.Info("Finished initializing state db")

	for i, tx := range executedBlock.Transactions() {
		if tx.Gas() > header.GasLimit {
			return nil, nil, fmt.Errorf("tx consumes %d gas, more than available in L1 block %d", tx.Gas(), header.GasLimit)
		}
		if tx.Gas() > uint64(*gasPool) {
			return nil, nil, fmt.Errorf("action takes too much gas: %d, only have %d", tx.Gas(), uint64(*gasPool))
		}
		statedb.SetTxContext(tx.Hash(), len(executedBlock.Transactions()))
		blockCtx := core.NewEVMBlockContext(header, chainCtx, nil, genesis.Config, statedb)
		evm := vm.NewEVM(blockCtx, statedb, genesis.Config, vm.Config{Tracer: blockTracer.Tracer()})
		_, err := core.ApplyTransaction(
			evm, gasPool, statedb, header, tx.WithoutBlobTxSidecar(), &header.GasUsed)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to apply transaction to L1 block (tx %d): %v", len(executedBlock.Transactions()), err)
		}

		prevBlockStats := blockStats.copy()
		blockStats.update(statedb, codes, blockTracer.opcodeStats, blockTracer.precompileStats)
		txStats[i] = blockStats.sub(prevBlockStats)
	}

	header.GasUsed = header.GasLimit - (uint64(*gasPool))
	header.Root = statedb.IntermediateRoot(true)

	log.Info("Finished executing block transactions")

	blockStats.update(statedb, codes, blockTracer.opcodeStats, blockTracer.precompileStats)

	isCancun := genesis.Config.IsCancun(header.Number, header.Time)
	// Write state changes to db
	root, err := statedb.Commit(header.Number.Uint64(), genesis.Config.IsEIP158(header.Number), isCancun)
	if err != nil {
		return nil, nil, fmt.Errorf("l1 state write error: %v", err)
	}
	if header.Root.Cmp(root) != 0 {
		return nil, nil, fmt.Errorf("l1 state root mismatch: %v != %v", root, header.Root)
	}

	log.Info("Finished committing state db")

	err = statedb.Database().TrieDB().Commit(root, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to commit state db: %w", err)
	}

	log.Info("Finished committing state db to trie db")

	return blockStats, txStats, nil
}
