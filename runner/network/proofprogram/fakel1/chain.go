package fakel1

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"os"

	"github.com/base/base-bench/runner/network/blocks"
	opEth "github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool/blobpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
)

type FakeL1Chain struct {
	chain *core.BlockChain

	l1Signer   types.Signer
	genesis    *core.Genesis
	log        log.Logger
	l1Database ethdb.Database

	l1BlobSidecars []*types.BlobTxSidecar
	blobStore      *BlobsStore
}

func (f *FakeL1Chain) GetNonce(addr common.Address) (uint64, error) {
	fmt.Printf("getting nonce for %s with latest block %d\n", addr.Hex(), f.chain.CurrentBlock().Number.Uint64())
	statedb, err := f.chain.State()
	if err != nil {
		return 0, err
	}
	nonce := statedb.GetNonce(addr)
	return nonce, nil
}

func (f *FakeL1Chain) BeaconGenesis() opEth.APIGenesisResponse {
	return opEth.APIGenesisResponse{
		Data: opEth.ReducedGenesisData{
			GenesisTime: opEth.Uint64String(f.genesis.Timestamp),
		},
	}
}
func (f *FakeL1Chain) ConfigSpec() opEth.APIConfigResponse {
	return opEth.APIConfigResponse{
		Data: opEth.ReducedConfigData{
			SecondsPerSlot: 1,
		},
	}
}

//type APIBlobSidecar struct {
// 	Index             Uint64String            `json:"index"`
// 	Blob              Blob                    `json:"blob"`
// 	KZGCommitment     Bytes48                 `json:"kzg_commitment"`
// 	KZGProof          Bytes48                 `json:"kzg_proof"`
// 	SignedBlockHeader SignedBeaconBlockHeader `json:"signed_block_header"`
// 	InclusionProof    []Bytes32               `json:"kzg_commitment_inclusion_proof"`
// }

func (f *FakeL1Chain) GetSidecarsBySlot(ctx context.Context, slot uint64) (*opEth.APIGetBlobSidecarsResponse, error) {
	slotTime := f.genesis.Timestamp + slot

	returnedSidecars, err := f.blobStore.GetAllSidecars(ctx, slotTime)
	if err != nil {
		return nil, err
	}
	sidecars := make([]*opEth.APIBlobSidecar, len(returnedSidecars))

	var mockBeaconBlockRoot [32]byte
	mockBeaconBlockRoot[0] = 42
	binary.LittleEndian.PutUint64(mockBeaconBlockRoot[32-8:], slot)

	fmt.Printf("returned sidecars for slot %d: %#v\n", slot, returnedSidecars)

	for i, sidecar := range returnedSidecars {
		sidecars[i] = &opEth.APIBlobSidecar{
			Index:         opEth.Uint64String(sidecar.Index),
			Blob:          sidecar.Blob,
			KZGCommitment: sidecar.KZGCommitment,
			KZGProof:      sidecar.KZGProof,
			SignedBlockHeader: opEth.SignedBeaconBlockHeader{
				Message: opEth.BeaconBlockHeader{
					StateRoot: mockBeaconBlockRoot,
					Slot:      opEth.Uint64String(slot),
				},
			},
			InclusionProof: make([]opEth.Bytes32, 0),
		}
	}

	return &opEth.APIGetBlobSidecarsResponse{
		Data: sidecars,
	}, nil
}

func (f *FakeL1Chain) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	return f.chain.GetBlockByHash(hash), nil
}
func (f *FakeL1Chain) GetBlockByNumber(number uint64) (*types.Block, error) {
	return f.chain.GetBlockByNumber(number), nil
}
func (f *FakeL1Chain) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return f.chain.GetReceiptsByHash(hash), nil
}

func (f *FakeL1Chain) PrintChain(log log.Logger) {
	currBlock := f.chain.CurrentBlock()
	for currBlock != nil {
		log.Info("Block", "number", currBlock.Number, "hash", currBlock.Hash().Hex())
		currBlock = f.chain.GetHeaderByHash(currBlock.ParentHash)
	}
}

func (f *FakeL1Chain) GetLatestBlock() (*types.Header, error) {
	return f.chain.CurrentBlock(), nil
}

func (f *FakeL1Chain) BuildAndMine(txs []*types.Transaction) error {
	parent := f.chain.CurrentBlock()
	header := &types.Header{
		ParentHash:      parent.Hash(),
		Coinbase:        parent.Coinbase,
		Difficulty:      common.Big0,
		Number:          new(big.Int).Add(parent.Number, common.Big1),
		GasLimit:        parent.GasLimit,
		Time:            parent.Time + 1,
		Extra:           parent.Extra,
		MixDigest:       common.Hash{}, // TODO: maybe randomize this (prev-randao value)
		WithdrawalsHash: &types.EmptyWithdrawalsHash,
	}

	statedb, err := state.New(parent.Root, state.NewDatabase(triedb.NewDatabase(f.l1Database, nil), nil))
	if err != nil {
		return fmt.Errorf("failed to init state db around block %s (state %s): %w", parent.Hash().Hex(), parent.Root.Hex(), err)
	}

	receipts := make([]*types.Receipt, 0)
	transactions := make([]*types.Transaction, 0)

	if f.genesis.Config.IsLondon(header.Number) {
		header.BaseFee = eip1559.CalcBaseFee(f.genesis.Config, parent, header.Time)
		// At the transition, double the gas limit so the gas target is equal to the old gas limit.
		if !f.genesis.Config.IsLondon(parent.Number) {
			header.GasLimit = parent.GasLimit * f.genesis.Config.ElasticityMultiplier()
		}
	}

	if f.genesis.Config.IsCancun(header.Number, header.Time) {
		header.BlobGasUsed = new(uint64)
		excessBlobGas := eip4844.CalcExcessBlobGas(f.genesis.Config, parent, header.Time)
		header.ExcessBlobGas = &excessBlobGas
		root := crypto.Keccak256Hash([]byte("fake-beacon-block-root"), header.Number.Bytes())
		header.ParentBeaconRoot = &root

		// Copied from op-program/client/l2/engineapi/block_processor.go
		// TODO(client-pod#826)
		// Unfortunately this is not part of any Geth environment setup,
		// we just have to apply it, like how the Geth block-builder worker does.
		context := core.NewEVMBlockContext(header, f.chain, nil, f.chain.Config(), statedb)
		// NOTE: Unlikely to be needed for the beacon block root, but we setup any precompile overrides anyways for forwards-compatibility
		var precompileOverrides vm.PrecompileOverrides
		if vmConfig := f.chain.GetVMConfig(); vmConfig != nil && vmConfig.PrecompileOverrides != nil {
			precompileOverrides = vmConfig.PrecompileOverrides
		}
		vmenv := vm.NewEVM(context, statedb, f.chain.Config(), vm.Config{PrecompileOverrides: precompileOverrides})
		core.ProcessBeaconBlockRoot(*header.ParentBeaconRoot, vmenv)

		if f.chain.Config().IsPrague(header.Number, header.Time) {
			core.ProcessParentBlockHash(header.ParentHash, vmenv)
		}
	}

	gasPool := new(core.GasPool)
	gasPool.AddGas(header.GasLimit)

	for _, tx := range txs {
		from, err := f.l1Signer.Sender(tx)
		if err != nil {
			return fmt.Errorf("failed to get sender of tx: %v", err)
		}
		f.log.Info("including tx", "nonce", tx.Nonce(), "from", from, "to", tx.To())
		if tx.Gas() > header.GasLimit {
			return fmt.Errorf("tx consumes %d gas, more than available in L1 block %d", tx.Gas(), header.GasLimit)
		}
		if tx.Gas() > uint64(*gasPool) {
			return fmt.Errorf("action takes too much gas: %d, only have %d", tx.Gas(), uint64(*gasPool))
		}
		statedb.SetTxContext(tx.Hash(), len(f.chain.GetBlockByHash(f.chain.CurrentBlock().Hash()).Transactions()))
		blockCtx := core.NewEVMBlockContext(header, f.chain, nil, f.chain.Config(), statedb)
		evm := vm.NewEVM(blockCtx, statedb, f.chain.Config(), *f.chain.GetVMConfig())
		receipt, err := core.ApplyTransaction(
			evm, gasPool, statedb, header, tx.WithoutBlobTxSidecar(), &header.GasUsed)
		if err != nil {
			return fmt.Errorf("failed to apply transaction to L1 block (tx %d): %v", len(f.chain.GetBlockByHash(f.chain.CurrentBlock().Hash()).Transactions()), err)
		}

		receipts = append(receipts, receipt)
		transactions = append(transactions, tx.WithoutBlobTxSidecar())

		if tx.Type() == types.BlobTxType {
			// require.True(t, s.l1Cfg.Config.IsCancun(s.l1BuildingHeader.Number, s.l1BuildingHeader.Time), "L1 must be cancun to process blob tx")
			if !f.chain.Config().IsCancun(header.Number, header.Time) {
				return fmt.Errorf("L1 must be cancun to process blob tx")
			}
			sidecar := tx.BlobTxSidecar()
			slot := (header.Time - f.genesis.Timestamp)
			log.Info("adding blob tx sidecar", "slot", slot, "number", header.Number, "blob_hashes", sidecar.BlobHashes())
			if sidecar != nil {
				f.l1BlobSidecars = append(f.l1BlobSidecars, sidecar)
				f.log.Info("added blob tx sidecar", "slot", slot, "number", header.Number, "blob_hashes", sidecar.BlobHashes())
			}
			*header.BlobGasUsed += receipt.BlobGasUsed
		}

	}

	for _, receipt := range receipts {
		fmt.Printf("receipt: %#v\n", receipt)
	}

	header.GasUsed = header.GasLimit - (uint64(*gasPool))
	header.Root = statedb.IntermediateRoot(true)

	isCancun := f.chain.Config().IsCancun(header.Number, header.Time)
	// Write state changes to db
	root, err := statedb.Commit(header.Number.Uint64(), f.chain.Config().IsEIP158(header.Number), isCancun)
	if err != nil {
		return fmt.Errorf("l1 state write error: %v", err)
	}
	if header.Root.Cmp(root) != 0 {
		return fmt.Errorf("l1 state root mismatch: %v != %v", root, header.Root)
	}

	block := types.NewBlock(header, &types.Body{
		Transactions: transactions,
		Withdrawals:  []*types.Withdrawal{},
	}, receipts, trie.NewStackTrie(nil), blocks.L1BlockType{})

	if err := statedb.Database().TrieDB().Commit(root, false); err != nil {
		return fmt.Errorf("l1 trie write error: %v", err)
	}
	// now that the blob txs are in a canonical block, flush them to the blob store
	i := 0
	for _, sidecar := range f.l1BlobSidecars {
		for idx, h := range sidecar.BlobHashes() {
			blob := (*opEth.Blob)(&sidecar.Blobs[idx])
			indexedHash := opEth.IndexedBlobHash{Index: uint64(i), Hash: h}
			f.blobStore.StoreBlob(block.Time(), indexedHash, blob)
			i++
		}
	}
	f.l1BlobSidecars = make([]*types.BlobTxSidecar, 0)
	_, err = f.chain.InsertChain(types.Blocks{block})
	if err != nil {
		return fmt.Errorf("failed to insert block into l1 chain: %v", err)
	}
	return nil
}

func NewFakeL1ChainWithGenesis(genesis *core.Genesis) (*FakeL1Chain, error) {
	// mkdir ./blobs if not exists
	// TODO: move to a nicer spot
	tempBlobDir := "./blobs"
	if err := os.MkdirAll(tempBlobDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create blobs directory: %w", err)
	}

	ethCfg := &ethconfig.Config{
		NetworkId:                 genesis.Config.ChainID.Uint64(),
		Genesis:                   genesis,
		RollupDisableTxPoolGossip: true,
		StateScheme:               rawdb.HashScheme,
		NoPruning:                 true,
		BlobPool: blobpool.Config{
			Datadir:   tempBlobDir,
			Datacap:   blobpool.DefaultConfig.Datacap,
			PriceBump: blobpool.DefaultConfig.PriceBump,
		},
	}
	nodeCfg := &node.Config{
		Name:        "l1-geth",
		WSHost:      "127.0.0.1",
		WSPort:      0,
		HTTPHost:    "127.0.0.1",
		HTTPPort:    0,
		WSModules:   []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal"},
		HTTPModules: []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal"},
		DataDir:     "", // in-memory
		P2P: p2p.Config{
			NoDiscovery: true,
			NoDial:      true,
		},
	}
	n, err := node.New(nodeCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	backend, err := eth.New(n, ethCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend: %w", err)
	}

	n.RegisterAPIs(tracers.APIs(backend.APIBackend))

	if err := n.Start(); err != nil {
		return nil, fmt.Errorf("failed to start L1 geth node: %w", err)
	}

	l1Chain := &FakeL1Chain{
		genesis:        genesis,
		blobStore:      NewBlobStore(),
		chain:          backend.BlockChain(),
		l1Signer:       types.NewPragueSigner(genesis.Config.ChainID),
		log:            log.New("chain", "l1"),
		l1Database:     backend.ChainDb(),
		l1BlobSidecars: make([]*types.BlobTxSidecar, 0),
	}

	err = l1Chain.BuildAndMine([]*types.Transaction{})
	if err != nil {
		return nil, fmt.Errorf("failed to build and mine genesis: %w", err)
	}

	return l1Chain, nil
}
