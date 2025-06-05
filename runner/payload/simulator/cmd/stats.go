package main

import (
	"context"
	"fmt"
	"math/big"

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
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

func fetchBlockStats(client *ethclient.Client, block *types.Block, genesis *core.Genesis) (map[string]interface{}, error) {
	var result *eth.ExecutionWitness
	err := client.Client().CallContext(context.Background(), &result, "debug_executionWitness", hexutil.EncodeUint64(block.NumberU64()))
	if err != nil {
		return nil, err
	}

	parentBlock, err := client.BlockByHash(context.Background(), block.ParentHash())
	if err != nil {
		return nil, err
	}

	err = executeBlock(client, parentBlock, block, result, genesis)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"blockNumber": block.Number(),
		"blockHash":   block.Hash(),
		"blockTime":   block.Time(),
		"blockSize":   block.Size(),
	}, nil
}

type blockCtx struct {
	engine                consensus.Engine
	getHeaderByHashNumber func(hash common.Hash, number uint64) *types.Header
	config                *params.ChainConfig
}

func newBlockCtx(genesis *core.Genesis, ethClient *ethclient.Client) *blockCtx {
	getHeaderByHashNumber := func(hash common.Hash, number uint64) *types.Header {
		header, err := ethClient.HeaderByNumber(context.Background(), nil)
		if err != nil {
			panic(err)
		}
		return header
	}

	return &blockCtx{
		engine:                beacon.New(nil),
		getHeaderByHashNumber: getHeaderByHashNumber,
		config:                genesis.Config,
	}
}

func (b *blockCtx) Engine() consensus.Engine {
	return b.engine
}

func (b *blockCtx) GetHeader(hash common.Hash, number uint64) *types.Header {
	return b.getHeaderByHashNumber(hash, number)
}

func (b *blockCtx) Config() *params.ChainConfig {
	return b.config
}

type stats struct {
	accountLoaded   float64
	accountDeleted  float64
	accountsUpdated float64
	storageLoaded   float64
	storageDeleted  float64
	storageUpdated  float64
}

func newStats() *stats {
	return &stats{
		accountLoaded:   0,
		accountDeleted:  0,
		accountsUpdated: 0,
		storageLoaded:   0,
		storageDeleted:  0,
		storageUpdated:  0,
	}
}

func (s *stats) update(db *state.StateDB) {
	s.accountLoaded = float64(db.AccountLoaded)
	s.accountDeleted = float64(db.AccountDeleted)
	s.accountsUpdated = float64(db.AccountUpdated)
	s.storageLoaded = float64(db.StorageLoaded)
	s.storageDeleted = float64(db.StorageDeleted.Load())
	s.storageUpdated = float64(db.StorageUpdated.Load())
}

func (s *stats) sub(other *stats) *stats {
	return &stats{
		accountLoaded:   s.accountLoaded - other.accountLoaded,
		accountDeleted:  s.accountDeleted - other.accountDeleted,
		accountsUpdated: s.accountsUpdated - other.accountsUpdated,
		storageLoaded:   s.storageLoaded - other.storageLoaded,
		storageDeleted:  s.storageDeleted - other.storageDeleted,
		storageUpdated:  s.storageUpdated - other.storageUpdated,
	}
}

func (s *stats) add(other *stats) *stats {
	return &stats{
		accountLoaded:   s.accountLoaded + other.accountLoaded,
		accountDeleted:  s.accountDeleted + other.accountDeleted,
		accountsUpdated: s.accountsUpdated + other.accountsUpdated,
		storageLoaded:   s.storageLoaded + other.storageLoaded,
		storageDeleted:  s.storageDeleted + other.storageDeleted,
		storageUpdated:  s.storageUpdated + other.storageUpdated,
	}
}

func (s *stats) mul(n float64) *stats {
	return &stats{
		accountLoaded:   s.accountLoaded * n,
		accountDeleted:  s.accountDeleted * n,
		accountsUpdated: s.accountsUpdated * n,
		storageLoaded:   s.storageLoaded * n,
		storageDeleted:  s.storageDeleted * n,
		storageUpdated:  s.storageUpdated * n,
	}
}

func (s *stats) copy() *stats {
	return &stats{
		accountLoaded:   s.accountLoaded,
		accountDeleted:  s.accountDeleted,
		accountsUpdated: s.accountsUpdated,
		storageLoaded:   s.storageLoaded,
		storageDeleted:  s.storageDeleted,
		storageUpdated:  s.storageUpdated,
	}
}

func (s *stats) String() string {
	return fmt.Sprintf("- Accounts Reads: %.0f\n- Accounts Deletes: %.0f\n- Accounts Updates: %.0f\n- Storage Reads: %.0f\n- Storage Deletes: %.0f\n- Storage Updates: %.0f\n", s.accountLoaded, s.accountDeleted, s.accountsUpdated, s.storageLoaded, s.storageDeleted, s.storageUpdated)
}

func executeBlock(client *ethclient.Client, parent *types.Block, executedBlock *types.Block, witness *eth.ExecutionWitness, genesis *core.Genesis) error {
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
		return fmt.Errorf("failed to get chain config: %w", err)
	}

	genesis.Config = chainCfg

	chainCtx := newBlockCtx(genesis, client)

	for _, code := range witness.Codes {
		codes[crypto.Keccak256Hash(code)] = []byte(code)
	}

	for _, node := range witness.State {
		nodes[crypto.Keccak256Hash(node)] = []byte(node)
	}

	db := memorydb.New()
	oracleKv := newPreimageOracle(db, codes, nodes)
	oracleDb := NewOracleBackedDB(db, oracleKv, eth.ChainIDFromBig(genesis.Config.ChainID))

	statedb, err := state.New(parent.Root(), state.NewDatabase(triedb.NewDatabase(rawdb.NewDatabase(oracleDb), nil), nil))
	if err != nil {
		return fmt.Errorf("failed to init state db around block %s (state %s): %w", parent.Hash().Hex(), parent.Root().Hex(), err)
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

	if genesis.Config.IsCancun(header.Number, header.Time) {
		header.BlobGasUsed = new(uint64)
		excessBlobGas := eip4844.CalcExcessBlobGas(genesis.Config, parent.Header(), header.Time)
		header.ExcessBlobGas = &excessBlobGas
		root := crypto.Keccak256Hash([]byte("fake-beacon-block-root"), header.Number.Bytes())
		header.ParentBeaconRoot = &root

		// Copied from op-program/client/l2/engineapi/block_processor.go
		// TODO(client-pod#826)
		// Unfortunately this is not part of any Geth environment setup,
		// we just have to apply it, like how the Geth block-builder worker does.
		context := core.NewEVMBlockContext(header, chainCtx, nil, genesis.Config, statedb)
		// NOTE: Unlikely to be needed for the beacon block root, but we setup any precompile overrides anyways for forwards-compatibility
		var precompileOverrides vm.PrecompileOverrides

		vmenv := vm.NewEVM(context, statedb, genesis.Config, vm.Config{PrecompileOverrides: precompileOverrides})
		core.ProcessBeaconBlockRoot(*header.ParentBeaconRoot, vmenv)

		if genesis.Config.IsPrague(header.Number, header.Time) {
			core.ProcessParentBlockHash(header.ParentHash, vmenv)
		}
	}

	gasPool := new(core.GasPool)
	gasPool.AddGas(header.GasLimit)

	blockStats.update(statedb)

	for i, tx := range executedBlock.Transactions() {
		if tx.Gas() > header.GasLimit {
			return fmt.Errorf("tx consumes %d gas, more than available in L1 block %d", tx.Gas(), header.GasLimit)
		}
		if tx.Gas() > uint64(*gasPool) {
			return fmt.Errorf("action takes too much gas: %d, only have %d", tx.Gas(), uint64(*gasPool))
		}
		statedb.SetTxContext(tx.Hash(), len(executedBlock.Transactions()))
		blockCtx := core.NewEVMBlockContext(header, chainCtx, nil, genesis.Config, statedb)
		evm := vm.NewEVM(blockCtx, statedb, genesis.Config, vm.Config{})
		_, err := core.ApplyTransaction(
			evm, gasPool, statedb, header, tx.WithoutBlobTxSidecar(), &header.GasUsed)
		if err != nil {
			return fmt.Errorf("failed to apply transaction to L1 block (tx %d): %v", len(executedBlock.Transactions()), err)
		}

		prevBlockStats := blockStats.copy()
		blockStats.update(statedb)
		txStats[i] = blockStats.sub(prevBlockStats)
	}

	header.GasUsed = header.GasLimit - (uint64(*gasPool))
	header.Root = statedb.IntermediateRoot(true)

	blockStats.update(statedb)

	isCancun := genesis.Config.IsCancun(header.Number, header.Time)
	// Write state changes to db
	root, err := statedb.Commit(header.Number.Uint64(), genesis.Config.IsEIP158(header.Number), isCancun)
	if err != nil {
		return fmt.Errorf("l1 state write error: %v", err)
	}
	if header.Root.Cmp(root) != 0 {
		return fmt.Errorf("l1 state root mismatch: %v != %v", root, header.Root)
	}

	err = statedb.Database().TrieDB().Commit(root, false)
	if err != nil {
		return fmt.Errorf("failed to commit state db: %w", err)
	}

	fmt.Printf("state root calculated: %s, state root in header: %s\n", root.Hex(), header.Root.Hex())

	fmt.Println("block stats")
	fmt.Println(blockStats)

	fmt.Println("tx stats")
	for i, txStat := range txStats {
		fmt.Printf("tx %d: %s\n", i, txStat)
	}

	return nil
}
