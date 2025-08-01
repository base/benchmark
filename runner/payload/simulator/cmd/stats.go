package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/base/base-bench/runner/payload/simulator/simulatorstats"
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

func fetchBlockStats(log log.Logger, client *ethclient.Client, block *types.Block, genesis *core.Genesis, headerCache map[common.Hash]*types.Header) (*simulatorstats.Stats, []*simulatorstats.Stats, error) {
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

func updateStats(db *state.StateDB, codePrestate map[common.Hash][]byte, s *simulatorstats.Stats) {
	s.AccountLoaded = float64(db.AccountLoaded)
	s.AccountDeleted = float64(db.AccountDeleted)
	s.AccountsUpdated = float64(db.AccountUpdated)
	s.StorageLoaded = float64(db.StorageLoaded)
	s.StorageDeleted = float64(db.StorageDeleted.Load())
	s.StorageUpdated = float64(db.StorageUpdated.Load())

	totalCodeSize := uint64(0)
	for _, code := range codePrestate {
		totalCodeSize += uint64(len(code))
	}

	s.CodeSizeLoaded = float64(totalCodeSize)
	s.NumContractsLoaded = float64(len(codePrestate))
	s.Opcodes = s.Opcodes.RemoveAllBut("EXP", "KECCAK256")
}
func executeBlock(log log.Logger, client *ethclient.Client, parent *types.Block, executedBlock *types.Block, witness *eth.ExecutionWitness, genesis *core.Genesis, headerCache map[common.Hash]*types.Header) (*simulatorstats.Stats, []*simulatorstats.Stats, error) {
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
	initialState, err := state.New(parent.Root(), state.NewDatabase(triedb.NewDatabase(rawdb.NewDatabase(oracleDb), nil), nil))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init state db around block %s (state %s): %w", parent.Hash().Hex(), parent.Root().Hex(), err)
	}
	blockTracer := newOpcodeTracer()
	statedb := initialState.Copy()

	blockStats := simulatorstats.NewStats()
	txStats := make([]*simulatorstats.Stats, len(executedBlock.Transactions()))

	if genesis.Config.IsLondon(header.Number) {
		header.BaseFee = eip1559.CalcBaseFee(genesis.Config, parent.Header(), header.Time)
		// At the transition, double the gas limit so the gas target is equal to the old gas limit.
		if !genesis.Config.IsLondon(parent.Number()) {
			header.GasLimit = parent.GasLimit() * genesis.Config.ElasticityMultiplier()
		}
	}

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

	updateStats(statedb, codes, blockStats)

	log.Info("Finished initializing state db")
	hookedState := state.NewHookedState(statedb, blockTracer.Tracer())

	for i, tx := range executedBlock.Transactions() {
		if tx.Gas() > header.GasLimit {
			return nil, nil, fmt.Errorf("tx consumes %d gas, more than available in L1 block %d", tx.Gas(), header.GasLimit)
		}
		if tx.Gas() > uint64(*gasPool) {
			return nil, nil, fmt.Errorf("action takes too much gas: %d, only have %d", tx.Gas(), uint64(*gasPool))
		}
		statedb.SetTxContext(tx.Hash(), len(executedBlock.Transactions()))
		blockCtx := core.NewEVMBlockContext(header, chainCtx, nil, genesis.Config, statedb)
		evm := vm.NewEVM(blockCtx, hookedState, genesis.Config, vm.Config{Tracer: blockTracer.Tracer()})
		_, err := core.ApplyTransaction(
			evm, gasPool, statedb, header, tx.WithoutBlobTxSidecar(), &header.GasUsed)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to apply transaction to L1 block (tx %d): %v", len(executedBlock.Transactions()), err)
		}

		prevBlockStats := blockStats.Copy()
		updateStats(statedb, codes, blockStats)
		blockStats.Precompiles = blockTracer.precompileStats.Copy()
		txStats[i] = blockStats.Sub(prevBlockStats)
	}

	header.GasUsed = header.GasLimit - (uint64(*gasPool))
	header.Root = statedb.IntermediateRoot(true)

	log.Info("Finished executing block transactions")

	updateStats(statedb, codes, blockStats)

	isCancun := genesis.Config.IsCancun(header.Number, header.Time)
	// Write state changes to db
	root, err := statedb.Commit(header.Number.Uint64(), genesis.Config.IsEIP158(header.Number), isCancun)
	if err != nil {
		return nil, nil, fmt.Errorf("l1 state write error: %v", err)
	}
	if header.Root.Cmp(root) != 0 {
		return nil, nil, fmt.Errorf("l1 state root mismatch: %v != %v", root, header.Root)
	}

	newAddresses := make([]common.Address, 0)
	for addr := range blockTracer.addressesChanged {
		if !initialState.Exist(addr) {
			newAddresses = append(newAddresses, addr)
		}
	}

	blockStats.AccountsCreated = float64(len(newAddresses))
	blockStats.AccountsUpdated -= float64(len(newAddresses))

	log.Info("Finished committing state db")

	err = statedb.Database().TrieDB().Commit(root, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to commit state db: %w", err)
	}

	log.Info("Finished committing state db to trie db")

	return blockStats, txStats, nil
}
