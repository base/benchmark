// Copyright 2015 The go-ethereum Authors
// Copyright 2025 Base
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package fakel1

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

// EthereumAPI provides an API to access Ethereum related information.
type EthereumAPI struct {
	b Backend
}

// NewEthereumAPI creates a new Ethereum protocol API.
func NewEthereumAPI(b Backend) *EthereumAPI {
	return &EthereumAPI{b}
}

// GasPrice returns a suggestion for a gas price for legacy transactions.
func (api *EthereumAPI) GasPrice(ctx context.Context) (*hexutil.Big, error) {
	tipcap, err := api.b.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}
	if head := api.b.CurrentHeader(); head.BaseFee != nil {
		tipcap.Add(tipcap, head.BaseFee)
	}
	return (*hexutil.Big)(tipcap), err
}

// MaxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic fee transactions.
func (api *EthereumAPI) MaxPriorityFeePerGas(ctx context.Context) (*hexutil.Big, error) {
	tipcap, err := api.b.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}
	return (*hexutil.Big)(tipcap), err
}

type feeHistoryResult struct {
	OldestBlock      *hexutil.Big     `json:"oldestBlock"`
	Reward           [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee          []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio     []float64        `json:"gasUsedRatio"`
	BlobBaseFee      []*hexutil.Big   `json:"baseFeePerBlobGas,omitempty"`
	BlobGasUsedRatio []float64        `json:"blobGasUsedRatio,omitempty"`
}

// FeeHistory returns the fee market history.
func (api *EthereumAPI) FeeHistory(ctx context.Context, blockCount math.HexOrDecimal64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*feeHistoryResult, error) {
	oldest, reward, baseFee, gasUsed, blobBaseFee, blobGasUsed, err := api.b.FeeHistory(ctx, uint64(blockCount), lastBlock, rewardPercentiles)
	if err != nil {
		return nil, err
	}
	results := &feeHistoryResult{
		OldestBlock:  (*hexutil.Big)(oldest),
		GasUsedRatio: gasUsed,
	}
	if reward != nil {
		results.Reward = make([][]*hexutil.Big, len(reward))
		for i, w := range reward {
			results.Reward[i] = make([]*hexutil.Big, len(w))
			for j, v := range w {
				results.Reward[i][j] = (*hexutil.Big)(v)
			}
		}
	}
	if baseFee != nil {
		results.BaseFee = make([]*hexutil.Big, len(baseFee))
		for i, v := range baseFee {
			results.BaseFee[i] = (*hexutil.Big)(v)
		}
	}
	if blobBaseFee != nil {
		results.BlobBaseFee = make([]*hexutil.Big, len(blobBaseFee))
		for i, v := range blobBaseFee {
			results.BlobBaseFee[i] = (*hexutil.Big)(v)
		}
	}
	if blobGasUsed != nil {
		results.BlobGasUsedRatio = blobGasUsed
	}
	return results, nil
}

// BlobBaseFee returns the base fee for blob gas at the current head.
func (api *EthereumAPI) BlobBaseFee(ctx context.Context) *hexutil.Big {
	return (*hexutil.Big)(api.b.BlobBaseFee(ctx))
}

// TxPoolAPI offers and API for the transaction pool. It only operates on data that is non-confidential.
type TxPoolAPI struct {
	b Backend
}

// NewTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewTxPoolAPI(b Backend) *TxPoolAPI {
	return &TxPoolAPI{b}
}

// Content returns the transactions contained within the transaction pool.
func (api *TxPoolAPI) Content() map[string]map[string]map[string]*RPCTransaction {
	content := map[string]map[string]map[string]*RPCTransaction{
		"pending": make(map[string]map[string]*RPCTransaction),
		"queued":  make(map[string]map[string]*RPCTransaction),
	}
	pending, queue := api.b.TxPoolContent()
	curHeader := api.b.CurrentHeader()
	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = NewRPCPendingTransaction(tx, curHeader, api.b.ChainConfig())
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = NewRPCPendingTransaction(tx, curHeader, api.b.ChainConfig())
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// ContentFrom returns the transactions contained within the transaction pool.
func (api *TxPoolAPI) ContentFrom(addr common.Address) map[string]map[string]*RPCTransaction {
	content := make(map[string]map[string]*RPCTransaction, 2)
	pending, queue := api.b.TxPoolContentFrom(addr)
	curHeader := api.b.CurrentHeader()

	// Build the pending transactions
	dump := make(map[string]*RPCTransaction, len(pending))
	for _, tx := range pending {
		dump[fmt.Sprintf("%d", tx.Nonce())] = NewRPCPendingTransaction(tx, curHeader, api.b.ChainConfig())
	}
	content["pending"] = dump

	// Build the queued transactions
	dump = make(map[string]*RPCTransaction, len(queue))
	for _, tx := range queue {
		dump[fmt.Sprintf("%d", tx.Nonce())] = NewRPCPendingTransaction(tx, curHeader, api.b.ChainConfig())
	}
	content["queued"] = dump

	return content
}

// Status returns the number of pending and queued transaction in the pool.
func (api *TxPoolAPI) Status() map[string]hexutil.Uint {
	pending, queue := api.b.Stats()
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(pending),
		"queued":  hexutil.Uint(queue),
	}
}

// Inspect retrieves the content of the transaction pool and flattens it into an
// easily inspectable list.
func (api *TxPoolAPI) Inspect() map[string]map[string]map[string]string {
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string),
		"queued":  make(map[string]map[string]string),
	}
	pending, queue := api.b.TxPoolContent()

	// Define a formatter to flatten a transaction into a string
	format := func(tx *types.Transaction) string {
		if to := tx.To(); to != nil {
			return fmt.Sprintf("%s: %v wei + %v gas × %v wei", tx.To().Hex(), tx.Value(), tx.Gas(), tx.GasPrice())
		}
		return fmt.Sprintf("contract creation: %v wei + %v gas × %v wei", tx.Value(), tx.Gas(), tx.GasPrice())
	}
	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// EthereumAccountAPI provides an API to access accounts managed by this node.
// It offers only methods that can retrieve accounts.
type EthereumAccountAPI struct {
	am *accounts.Manager
}

// NewEthereumAccountAPI creates a new EthereumAccountAPI.
func NewEthereumAccountAPI(am *accounts.Manager) *EthereumAccountAPI {
	return &EthereumAccountAPI{am: am}
}

// Accounts returns the collection of accounts this node manages.
func (api *EthereumAccountAPI) Accounts() []common.Address {
	return api.am.Accounts()
}

// BlockChainAPI provides an API to access Ethereum blockchain data.
type BlockChainAPI struct {
	b Backend
}

// NewBlockChainAPI creates a new Ethereum blockchain API.
func NewBlockChainAPI(b Backend) *BlockChainAPI {
	return &BlockChainAPI{b}
}

// ChainId is the EIP-155 replay-protection chain id for the current Ethereum chain config.
//
// Note, this method does not conform to EIP-695 because the configured chain ID is always
// returned, regardless of the current head block. We used to return an error when the chain
// wasn't synced up to a block where EIP-155 is enabled, but this behavior caused issues
// in CL clients.
func (api *BlockChainAPI) ChainId() *hexutil.Big {
	return (*hexutil.Big)(api.b.ChainConfig().ChainID)
}

// BlockNumber returns the block number of the chain head.
func (api *BlockChainAPI) BlockNumber() hexutil.Uint64 {
	header, _ := api.b.HeaderByNumber(context.Background(), rpc.LatestBlockNumber) // latest header should always be available
	return hexutil.Uint64(header.Number.Uint64())
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (api *BlockChainAPI) GetBalance(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	header, err := headerByNumberOrHash(ctx, api.b, blockNrOrHash)
	if err != nil {
		return nil, err
	}

	if api.b.ChainConfig().IsOptimismPreBedrock(header.Number) {
		if api.b.HistoricalRPCService() != nil {
			var res hexutil.Big
			err := api.b.HistoricalRPCService().CallContext(ctx, &res, "eth_getBalance", address, blockNrOrHash)
			if err != nil {
				return nil, fmt.Errorf("historical backend error: %w", err)
			}
			return &res, nil
		} else {
			return nil, rpc.ErrNoHistoricalFallback
		}
	}

	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	b := state.GetBalance(address).ToBig()
	return (*hexutil.Big)(b), state.Error()
}

// AccountResult structs for GetProof
type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// proofList implements ethdb.KeyValueWriter and collects the proofs as
// hex-strings for delivery to rpc-caller.
type proofList []string

func (n *proofList) Put(key []byte, value []byte) error {
	*n = append(*n, hexutil.Encode(value))
	return nil
}

func (n *proofList) Delete(key []byte) error {
	panic("not supported")
}

// GetProof returns the Merkle-proof for a given account and optionally some storage keys.
func (api *BlockChainAPI) GetProof(ctx context.Context, address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (*AccountResult, error) {
	header, err := headerByNumberOrHash(ctx, api.b, blockNrOrHash)
	if err != nil {
		return nil, err
	}
	if api.b.ChainConfig().IsOptimismPreBedrock(header.Number) {
		if api.b.HistoricalRPCService() != nil {
			var res AccountResult
			err := api.b.HistoricalRPCService().CallContext(ctx, &res, "eth_getProof", address, storageKeys, blockNrOrHash)
			if err != nil {
				return nil, fmt.Errorf("historical backend error: %w", err)
			}
			return &res, nil
		} else {
			return nil, rpc.ErrNoHistoricalFallback
		}
	}
	var (
		keys         = make([]common.Hash, len(storageKeys))
		keyLengths   = make([]int, len(storageKeys))
		storageProof = make([]StorageResult, len(storageKeys))
	)
	// Deserialize all keys. This prevents state access on invalid input.
	for i, hexKey := range storageKeys {
		var err error
		keys[i], keyLengths[i], err = decodeHash(hexKey)
		if err != nil {
			return nil, err
		}
	}
	statedb, header, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if statedb == nil || err != nil {
		return nil, err
	}
	codeHash := statedb.GetCodeHash(address)
	storageRoot := statedb.GetStorageRoot(address)

	if len(keys) > 0 {
		var storageTrie state.Trie
		if storageRoot != types.EmptyRootHash && storageRoot != (common.Hash{}) {
			id := trie.StorageTrieID(header.Root, crypto.Keccak256Hash(address.Bytes()), storageRoot)
			st, err := trie.NewStateTrie(id, statedb.Database().TrieDB())
			if err != nil {
				return nil, err
			}
			storageTrie = st
		}
		// Create the proofs for the storageKeys.
		for i, key := range keys {
			// Output key encoding is a bit special: if the input was a 32-byte hash, it is
			// returned as such. Otherwise, we apply the QUANTITY encoding mandated by the
			// JSON-RPC spec for getProof. This behavior exists to preserve backwards
			// compatibility with older client versions.
			var outputKey string
			if keyLengths[i] != 32 {
				outputKey = hexutil.EncodeBig(key.Big())
			} else {
				outputKey = hexutil.Encode(key[:])
			}
			if storageTrie == nil {
				storageProof[i] = StorageResult{outputKey, &hexutil.Big{}, []string{}}
				continue
			}
			var proof proofList
			if err := storageTrie.Prove(crypto.Keccak256(key.Bytes()), &proof); err != nil {
				return nil, err
			}
			value := (*hexutil.Big)(statedb.GetState(address, key).Big())
			storageProof[i] = StorageResult{outputKey, value, proof}
		}
	}
	// Create the accountProof.
	tr, err := trie.NewStateTrie(trie.StateTrieID(header.Root), statedb.Database().TrieDB())
	if err != nil {
		return nil, err
	}
	var accountProof proofList
	if err := tr.Prove(crypto.Keccak256(address.Bytes()), &accountProof); err != nil {
		return nil, err
	}
	balance := statedb.GetBalance(address).ToBig()
	return &AccountResult{
		Address:      address,
		AccountProof: accountProof,
		Balance:      (*hexutil.Big)(balance),
		CodeHash:     codeHash,
		Nonce:        hexutil.Uint64(statedb.GetNonce(address)),
		StorageHash:  storageRoot,
		StorageProof: storageProof,
	}, statedb.Error()
}

// decodeHash parses a hex-encoded 32-byte hash. The input may optionally
// be prefixed by 0x and can have a byte length up to 32.
func decodeHash(s string) (h common.Hash, inputLength int, err error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	if (len(s) & 1) > 0 {
		s = "0" + s
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return common.Hash{}, 0, errors.New("hex string invalid")
	}
	if len(b) > 32 {
		return common.Hash{}, len(b), errors.New("hex string too long, want at most 32 bytes")
	}
	return common.BytesToHash(b), len(b), nil
}

// GetHeaderByNumber returns the requested canonical block header.
//   - When blockNr is -1 the chain pending header is returned.
//   - When blockNr is -2 the chain latest header is returned.
//   - When blockNr is -3 the chain finalized header is returned.
//   - When blockNr is -4 the chain safe header is returned.
func (api *BlockChainAPI) GetHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (map[string]interface{}, error) {
	header, err := api.b.HeaderByNumber(ctx, number)
	if header != nil && err == nil {
		response := RPCMarshalHeader(header)
		if number == rpc.PendingBlockNumber && api.b.ChainConfig().Optimism == nil {
			// Pending header need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

// GetHeaderByHash returns the requested header by hash.
func (api *BlockChainAPI) GetHeaderByHash(ctx context.Context, hash common.Hash) map[string]interface{} {
	header, _ := api.b.HeaderByHash(ctx, hash)
	if header != nil {
		return RPCMarshalHeader(header)
	}
	return nil
}

// GetBlockByNumber returns the requested canonical block.
//   - When blockNr is -1 the chain pending block is returned.
//   - When blockNr is -2 the chain latest block is returned.
//   - When blockNr is -3 the chain finalized block is returned.
//   - When blockNr is -4 the chain safe block is returned.
//   - When fullTx is true all transactions in the block are returned, otherwise
//     only the transaction hash is returned.
func (api *BlockChainAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := api.b.BlockByNumber(ctx, number)
	if block != nil && err == nil {
		response, err := RPCMarshalBlock(ctx, block, true, fullTx, api.b.ChainConfig(), api.b)
		if err == nil && number == rpc.PendingBlockNumber && api.b.ChainConfig().Optimism == nil {
			// Pending blocks need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (api *BlockChainAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := api.b.BlockByHash(ctx, hash)
	if block != nil {
		return RPCMarshalBlock(ctx, block, true, fullTx, api.b.ChainConfig(), api.b)
	}
	return nil, err
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block hash and index.
func (api *BlockChainAPI) GetUncleByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := api.b.BlockByNumber(ctx, blockNr)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			log.Debug("Requested uncle not found", "number", blockNr, "hash", block.Hash(), "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return RPCMarshalBlock(ctx, block, false, false, api.b.ChainConfig(), api.b)
	}
	return nil, err
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index.
func (api *BlockChainAPI) GetUncleByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := api.b.BlockByHash(ctx, blockHash)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			log.Debug("Requested uncle not found", "number", block.Number(), "hash", blockHash, "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return RPCMarshalBlock(ctx, block, false, false, api.b.ChainConfig(), api.b)
	}
	return nil, err
}

// GetUncleCountByBlockNumber returns number of uncles in the block for the given block number
func (api *BlockChainAPI) GetUncleCountByBlockNumber(ctx context.Context, blockNr rpc.BlockNumber) *hexutil.Uint {
	if block, _ := api.b.BlockByNumber(ctx, blockNr); block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n
	}
	return nil
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
func (api *BlockChainAPI) GetUncleCountByBlockHash(ctx context.Context, blockHash common.Hash) *hexutil.Uint {
	if block, _ := api.b.BlockByHash(ctx, blockHash); block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n
	}
	return nil
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (api *BlockChainAPI) GetCode(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	header, err := headerByNumberOrHash(ctx, api.b, blockNrOrHash)
	if err != nil {
		return nil, err
	}

	if api.b.ChainConfig().IsOptimismPreBedrock(header.Number) {
		if api.b.HistoricalRPCService() != nil {
			var res hexutil.Bytes
			err := api.b.HistoricalRPCService().CallContext(ctx, &res, "eth_getCode", address, blockNrOrHash)
			if err != nil {
				return nil, fmt.Errorf("historical backend error: %w", err)
			}
			return res, nil
		} else {
			return nil, rpc.ErrNoHistoricalFallback
		}
	}

	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}

	code := state.GetCode(address)
	return code, state.Error()
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block
// numbers are also allowed.
func (api *BlockChainAPI) GetStorageAt(ctx context.Context, address common.Address, hexKey string, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	header, err := headerByNumberOrHash(ctx, api.b, blockNrOrHash)
	if err != nil {
		return nil, err
	}

	if api.b.ChainConfig().IsOptimismPreBedrock(header.Number) {
		if api.b.HistoricalRPCService() != nil {
			var res hexutil.Bytes
			err := api.b.HistoricalRPCService().CallContext(ctx, &res, "eth_getStorageAt", address, hexKey, blockNrOrHash)
			if err != nil {
				return nil, fmt.Errorf("historical backend error: %w", err)
			}
			return res, nil
		} else {
			return nil, rpc.ErrNoHistoricalFallback
		}
	}

	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}

	key, _, err := decodeHash(hexKey)
	if err != nil {
		return nil, fmt.Errorf("unable to decode storage key: %s", err)
	}
	res := state.GetState(address, key)
	return res[:], state.Error()
}

// The HeaderByNumberOrHash method returns a nil error and nil header
// if the header is not found, but only for nonexistent block numbers. This is
// different from StateAndHeaderByNumberOrHash. To account for this discrepancy,
// headerOrNumberByHash will properly convert the error into an ethereum.NotFound.
func headerByNumberOrHash(ctx context.Context, b Backend, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	header, err := b.HeaderByNumberOrHash(ctx, blockNrOrHash)
	if header == nil {
		return nil, fmt.Errorf("header %w", ethereum.NotFound)
	}
	return header, err
}

// GetBlockReceipts returns the block receipts for the given block hash or number or tag.
func (api *BlockChainAPI) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]map[string]interface{}, error) {
	block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
	if block == nil || err != nil {
		// When the block doesn't exist, the RPC method should return JSON null
		// as per specification.
		return nil, nil
	}
	receipts, err := api.b.GetReceipts(ctx, block.Hash())
	if err != nil {
		return nil, err
	}
	txs := block.Transactions()
	if len(txs) != len(receipts) {
		return nil, fmt.Errorf("receipts length mismatch: %d vs %d", len(txs), len(receipts))
	}

	// Derive the sender.
	signer := types.MakeSigner(api.b.ChainConfig(), block.Number(), block.Time())

	result := make([]map[string]interface{}, len(receipts))
	for i, receipt := range receipts {
		result[i] = marshalReceipt(receipt, block.Hash(), block.NumberU64(), signer, txs[i], i, api.b.ChainConfig())
	}

	return result, nil
}

// ChainContextBackend provides methods required to implement ChainContext.
type ChainContextBackend interface {
	Engine() consensus.Engine
	HeaderByNumber(context.Context, rpc.BlockNumber) (*types.Header, error)
	ChainConfig() *params.ChainConfig
}

// ChainContext is an implementation of core.ChainContext. It's main use-case
// is instantiating a vm.BlockContext without having access to the BlockChain object.
type ChainContext struct {
	b   ChainContextBackend
	ctx context.Context
}

// NewChainContext creates a new ChainContext object.
func NewChainContext(ctx context.Context, backend ChainContextBackend) *ChainContext {
	return &ChainContext{ctx: ctx, b: backend}
}

func (context *ChainContext) Engine() consensus.Engine {
	return context.b.Engine()
}

func (context *ChainContext) GetHeader(hash common.Hash, number uint64) *types.Header {
	// This method is called to get the hash for a block number when executing the BLOCKHASH
	// opcode. Hence no need to search for non-canonical blocks.
	header, err := context.b.HeaderByNumber(context.ctx, rpc.BlockNumber(number))
	if err != nil || header.Hash() != hash {
		return nil
	}
	return header
}

func (context *ChainContext) Config() *params.ChainConfig {
	return context.b.ChainConfig()
}

// RPCMarshalHeader converts the given header to the RPC output .
func RPCMarshalHeader(head *types.Header) map[string]interface{} {
	result := map[string]interface{}{
		"number":           (*hexutil.Big)(head.Number),
		"hash":             head.Hash(),
		"parentHash":       head.ParentHash,
		"nonce":            head.Nonce,
		"mixHash":          head.MixDigest,
		"sha3Uncles":       head.UncleHash,
		"logsBloom":        head.Bloom,
		"stateRoot":        head.Root,
		"miner":            head.Coinbase,
		"difficulty":       (*hexutil.Big)(head.Difficulty),
		"extraData":        hexutil.Bytes(head.Extra),
		"gasLimit":         hexutil.Uint64(head.GasLimit),
		"gasUsed":          hexutil.Uint64(head.GasUsed),
		"timestamp":        hexutil.Uint64(head.Time),
		"transactionsRoot": head.TxHash,
		"receiptsRoot":     head.ReceiptHash,
	}
	if head.BaseFee != nil {
		result["baseFeePerGas"] = (*hexutil.Big)(head.BaseFee)
	}
	if head.WithdrawalsHash != nil {
		result["withdrawalsRoot"] = head.WithdrawalsHash
	}
	if head.BlobGasUsed != nil {
		result["blobGasUsed"] = hexutil.Uint64(*head.BlobGasUsed)
	}
	if head.ExcessBlobGas != nil {
		result["excessBlobGas"] = hexutil.Uint64(*head.ExcessBlobGas)
	}
	if head.ParentBeaconRoot != nil {
		result["parentBeaconBlockRoot"] = head.ParentBeaconRoot
	}
	if head.RequestsHash != nil {
		result["requestsHash"] = head.RequestsHash
	}
	return result
}

// RPCMarshalBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func RPCMarshalBlock(ctx context.Context, block *types.Block, inclTx bool, fullTx bool, config *params.ChainConfig, backend ReceiptGetter) (map[string]interface{}, error) {
	fields := RPCMarshalHeader(block.Header())
	fields["size"] = hexutil.Uint64(block.Size())

	if inclTx {
		formatTx := func(idx int, tx *types.Transaction) interface{} {
			return tx.Hash()
		}
		if fullTx {
			formatTx = func(idx int, _ *types.Transaction) interface{} {
				return newRPCTransactionFromBlockIndex(ctx, block, uint64(idx), config, backend)
			}
		}
		txs := block.Transactions()
		transactions := make([]interface{}, len(txs))
		for i, tx := range txs {
			transactions[i] = formatTx(i, tx)
		}
		fields["transactions"] = transactions
	}
	uncles := block.Uncles()
	uncleHashes := make([]common.Hash, len(uncles))
	for i, uncle := range uncles {
		uncleHashes[i] = uncle.Hash()
	}
	fields["uncles"] = uncleHashes
	if block.Withdrawals() != nil {
		fields["withdrawals"] = block.Withdrawals()
	}
	return fields, nil
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash           *common.Hash                 `json:"blockHash"`
	BlockNumber         *hexutil.Big                 `json:"blockNumber"`
	From                common.Address               `json:"from"`
	Gas                 hexutil.Uint64               `json:"gas"`
	GasPrice            *hexutil.Big                 `json:"gasPrice"`
	GasFeeCap           *hexutil.Big                 `json:"maxFeePerGas,omitempty"`
	GasTipCap           *hexutil.Big                 `json:"maxPriorityFeePerGas,omitempty"`
	MaxFeePerBlobGas    *hexutil.Big                 `json:"maxFeePerBlobGas,omitempty"`
	Hash                common.Hash                  `json:"hash"`
	Input               hexutil.Bytes                `json:"input"`
	Nonce               hexutil.Uint64               `json:"nonce"`
	To                  *common.Address              `json:"to"`
	TransactionIndex    *hexutil.Uint64              `json:"transactionIndex"`
	Value               *hexutil.Big                 `json:"value"`
	Type                hexutil.Uint64               `json:"type"`
	Accesses            *types.AccessList            `json:"accessList,omitempty"`
	ChainID             *hexutil.Big                 `json:"chainId,omitempty"`
	BlobVersionedHashes []common.Hash                `json:"blobVersionedHashes,omitempty"`
	AuthorizationList   []types.SetCodeAuthorization `json:"authorizationList,omitempty"`
	V                   *hexutil.Big                 `json:"v"`
	R                   *hexutil.Big                 `json:"r"`
	S                   *hexutil.Big                 `json:"s"`
	YParity             *hexutil.Uint64              `json:"yParity,omitempty"`

	// deposit-tx only
	SourceHash *common.Hash `json:"sourceHash,omitempty"`
	Mint       *hexutil.Big `json:"mint,omitempty"`
	IsSystemTx *bool        `json:"isSystemTx,omitempty"`
	// deposit-tx post-Canyon only
	DepositReceiptVersion *hexutil.Uint64 `json:"depositReceiptVersion,omitempty"`
}

// newRPCTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func newRPCTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber uint64, blockTime uint64, index uint64, baseFee *big.Int, config *params.ChainConfig, receipt *types.Receipt) *RPCTransaction {
	signer := types.MakeSigner(config, new(big.Int).SetUint64(blockNumber), blockTime)
	from, _ := types.Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()
	result := &RPCTransaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}

	switch tx.Type() {
	case types.DepositTxType:
		srcHash := tx.SourceHash()
		isSystemTx := tx.IsSystemTx()
		result.SourceHash = &srcHash
		if isSystemTx {
			// Only include IsSystemTx when true
			result.IsSystemTx = &isSystemTx
		}
		result.Mint = (*hexutil.Big)(tx.Mint())
		if receipt != nil && receipt.DepositNonce != nil {
			result.Nonce = hexutil.Uint64(*receipt.DepositNonce)
			if receipt.DepositReceiptVersion != nil {
				result.DepositReceiptVersion = new(hexutil.Uint64)
				*result.DepositReceiptVersion = hexutil.Uint64(*receipt.DepositReceiptVersion)
			}
		}
	case types.LegacyTxType:
		if v.Sign() == 0 && r.Sign() == 0 && s.Sign() == 0 { // pre-bedrock relayed tx does not have a signature
			result.ChainID = (*hexutil.Big)(new(big.Int).Set(config.ChainID))
			break
		}
		// if a legacy transaction has an EIP-155 chain id, include it explicitly
		if id := tx.ChainId(); id.Sign() != 0 {
			result.ChainID = (*hexutil.Big)(id)
		}

	case types.AccessListTxType:
		al := tx.AccessList()
		yparity := hexutil.Uint64(v.Sign())
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.YParity = &yparity

	case types.DynamicFeeTxType:
		al := tx.AccessList()
		yparity := hexutil.Uint64(v.Sign())
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.YParity = &yparity
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (common.Hash{}) {
			// price = min(gasTipCap + baseFee, gasFeeCap)
			result.GasPrice = (*hexutil.Big)(effectiveGasPrice(tx, baseFee))
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}

	case types.BlobTxType:
		al := tx.AccessList()
		yparity := hexutil.Uint64(v.Sign())
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.YParity = &yparity
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (common.Hash{}) {
			result.GasPrice = (*hexutil.Big)(effectiveGasPrice(tx, baseFee))
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
		result.MaxFeePerBlobGas = (*hexutil.Big)(tx.BlobGasFeeCap())
		result.BlobVersionedHashes = tx.BlobHashes()

	case types.SetCodeTxType:
		al := tx.AccessList()
		yparity := hexutil.Uint64(v.Sign())
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.YParity = &yparity
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (common.Hash{}) {
			result.GasPrice = (*hexutil.Big)(effectiveGasPrice(tx, baseFee))
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
		result.AuthorizationList = tx.SetCodeAuthorizations()
	}
	return result
}

// effectiveGasPrice computes the transaction gas fee, based on the given basefee value.
//
//	price = min(gasTipCap + baseFee, gasFeeCap)
func effectiveGasPrice(tx *types.Transaction, baseFee *big.Int) *big.Int {
	fee := tx.GasTipCap()
	fee = fee.Add(fee, baseFee)
	if tx.GasFeeCapIntCmp(fee) < 0 {
		return tx.GasFeeCap()
	}
	return fee
}

// NewRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func NewRPCPendingTransaction(tx *types.Transaction, current *types.Header, config *params.ChainConfig) *RPCTransaction {
	var (
		baseFee     *big.Int
		blockNumber = uint64(0)
		blockTime   = uint64(0)
	)
	if current != nil {
		baseFee = eip1559.CalcBaseFee(config, current, current.Time+1)
		blockNumber = current.Number.Uint64()
		blockTime = current.Time
	}
	return newRPCTransaction(tx, common.Hash{}, blockNumber, blockTime, 0, baseFee, config, nil)
}

// newRPCTransactionFromBlockIndex returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(ctx context.Context, b *types.Block, index uint64, config *params.ChainConfig, backend ReceiptGetter) *RPCTransaction {
	txs := b.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	tx := txs[index]
	rcpt := depositTxReceipt(ctx, b.Hash(), index, backend, tx)
	return newRPCTransaction(tx, b.Hash(), b.NumberU64(), b.Time(), index, b.BaseFee(), config, rcpt)
}

type ReceiptGetter interface {
	GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error)
}

func depositTxReceipt(ctx context.Context, blockHash common.Hash, index uint64, backend ReceiptGetter, tx *types.Transaction) *types.Receipt {
	if tx.Type() != types.DepositTxType {
		return nil
	}
	receipts, err := backend.GetReceipts(ctx, blockHash)
	if err != nil {
		return nil
	}
	if index >= uint64(len(receipts)) {
		return nil
	}
	return receipts[index]
}

// marshalReceipt marshals a transaction receipt into a JSON object.
func marshalReceipt(receipt *types.Receipt, blockHash common.Hash, blockNumber uint64, signer types.Signer, tx *types.Transaction, txIndex int, chainConfig *params.ChainConfig) map[string]interface{} {
	from, _ := types.Sender(signer, tx)

	fields := map[string]interface{}{
		"blockHash":         blockHash,
		"blockNumber":       hexutil.Uint64(blockNumber),
		"transactionHash":   tx.Hash(),
		"transactionIndex":  hexutil.Uint64(txIndex),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		"logsBloom":         receipt.Bloom,
		"type":              hexutil.Uint(tx.Type()),
		"effectiveGasPrice": (*hexutil.Big)(receipt.EffectiveGasPrice),
	}

	if chainConfig.Optimism != nil && !tx.IsDepositTx() {
		fields["l1GasPrice"] = (*hexutil.Big)(receipt.L1GasPrice)
		fields["l1GasUsed"] = (*hexutil.Big)(receipt.L1GasUsed)
		fields["l1Fee"] = (*hexutil.Big)(receipt.L1Fee)
		// Fields removed with Ecotone
		if receipt.FeeScalar != nil {
			fields["l1FeeScalar"] = receipt.FeeScalar.String()
		}
		// Fields added in Ecotone
		if receipt.L1BlobBaseFee != nil {
			fields["l1BlobBaseFee"] = (*hexutil.Big)(receipt.L1BlobBaseFee)
		}
		if receipt.L1BaseFeeScalar != nil {
			fields["l1BaseFeeScalar"] = hexutil.Uint64(*receipt.L1BaseFeeScalar)
		}
		if receipt.L1BlobBaseFeeScalar != nil {
			fields["l1BlobBaseFeeScalar"] = hexutil.Uint64(*receipt.L1BlobBaseFeeScalar)
		}
		// Fields added in Isthmus
		if receipt.OperatorFeeScalar != nil {
			fields["operatorFeeScalar"] = hexutil.Uint64(*receipt.OperatorFeeScalar)
		}
		if receipt.OperatorFeeConstant != nil {
			fields["operatorFeeConstant"] = hexutil.Uint64(*receipt.OperatorFeeConstant)
		}
	}
	if chainConfig.Optimism != nil && tx.IsDepositTx() && receipt.DepositNonce != nil {
		fields["depositNonce"] = hexutil.Uint64(*receipt.DepositNonce)
		if receipt.DepositReceiptVersion != nil {
			fields["depositReceiptVersion"] = hexutil.Uint64(*receipt.DepositReceiptVersion)
		}
	}

	// Assign receipt status or post state.
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = []*types.Log{}
	}

	if tx.Type() == types.BlobTxType {
		fields["blobGasUsed"] = hexutil.Uint64(receipt.BlobGasUsed)
		fields["blobGasPrice"] = (*hexutil.Big)(receipt.BlobGasPrice)
	}

	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields
}

// SubmitTransaction is a helper function that submits tx to txPool and logs a message.
func SubmitTransaction(ctx context.Context, b Backend, tx *types.Transaction) (common.Hash, error) {
	// If the transaction fee cap is already specified, ensure the
	// fee of the given transaction is _reasonable_.
	if err := checkTxFee(tx.GasPrice(), tx.Gas(), b.RPCTxFeeCap()); err != nil {
		return common.Hash{}, err
	}
	if !b.UnprotectedAllowed() && !tx.Protected() {
		// Ensure only eip155 signed transactions are submitted if EIP155Required is set.
		return common.Hash{}, errors.New("only replay-protected (EIP-155) transactions allowed over RPC")
	}
	if err := b.SendTx(ctx, tx); err != nil {
		return common.Hash{}, err
	}
	// Print a log with full tx details for manual investigations and interventions
	head := b.CurrentBlock()
	signer := types.MakeSigner(b.ChainConfig(), head.Number, head.Time)
	from, err := types.Sender(signer, tx)
	if err != nil {
		return common.Hash{}, err
	}

	if tx.To() == nil {
		addr := crypto.CreateAddress(from, tx.Nonce())
		log.Info("Submitted contract creation", "hash", tx.Hash().Hex(), "from", from, "nonce", tx.Nonce(), "contract", addr.Hex(), "value", tx.Value())
	} else {
		log.Info("Submitted transaction", "hash", tx.Hash().Hex(), "from", from, "nonce", tx.Nonce(), "recipient", tx.To(), "value", tx.Value())
	}
	return tx.Hash(), nil
}

// DebugAPI is the collection of Ethereum APIs exposed over the debugging
// namespace.
type DebugAPI struct {
	b Backend
}

// NewDebugAPI creates a new instance of DebugAPI.
func NewDebugAPI(b Backend) *DebugAPI {
	return &DebugAPI{b: b}
}

// GetRawHeader retrieves the RLP encoding for a single header.
func (api *DebugAPI) GetRawHeader(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	var hash common.Hash
	if h, ok := blockNrOrHash.Hash(); ok {
		hash = h
	} else {
		block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if block == nil || err != nil {
			return nil, err
		}
		hash = block.Hash()
	}
	header, _ := api.b.HeaderByHash(ctx, hash)
	if header == nil {
		return nil, fmt.Errorf("header #%d not found", hash)
	}
	return rlp.EncodeToBytes(header)
}

// GetRawBlock retrieves the RLP encoded for a single block.
func (api *DebugAPI) GetRawBlock(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	var hash common.Hash
	if h, ok := blockNrOrHash.Hash(); ok {
		hash = h
	} else {
		block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if block == nil || err != nil {
			return nil, err
		}
		hash = block.Hash()
	}
	block, _ := api.b.BlockByHash(ctx, hash)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", hash)
	}
	return rlp.EncodeToBytes(block)
}

// GetRawReceipts retrieves the binary-encoded receipts of a single block.
func (api *DebugAPI) GetRawReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]hexutil.Bytes, error) {
	var hash common.Hash
	if h, ok := blockNrOrHash.Hash(); ok {
		hash = h
	} else {
		block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if block == nil || err != nil {
			return nil, err
		}
		hash = block.Hash()
	}
	receipts, err := api.b.GetReceipts(ctx, hash)
	if err != nil {
		return nil, err
	}
	result := make([]hexutil.Bytes, len(receipts))
	for i, receipt := range receipts {
		b, err := receipt.MarshalBinary()
		if err != nil {
			return nil, err
		}
		result[i] = b
	}
	return result, nil
}

// GetRawTransaction returns the bytes of the transaction for the given hash.
func (api *DebugAPI) GetRawTransaction(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	// Retrieve a finalized transaction, or a pooled otherwise
	found, tx, _, _, _, err := api.b.GetTransaction(ctx, hash)
	if !found {
		if tx = api.b.GetPoolTransaction(hash); tx != nil {
			return tx.MarshalBinary()
		}
		if err == nil {
			return nil, nil
		}
		return nil, err
	}
	return tx.MarshalBinary()
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (api *DebugAPI) PrintBlock(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return spew.Sdump(block), nil
}

// ChaindbProperty returns leveldb properties of the key-value database.
func (api *DebugAPI) ChaindbProperty() (string, error) {
	return api.b.ChainDb().Stat()
}

// ChaindbCompact flattens the entire key-value database into a single level,
// removing all unused slots and merging all keys.
func (api *DebugAPI) ChaindbCompact() error {
	cstart := time.Now()
	for b := 0; b <= 255; b++ {
		var (
			start = []byte{byte(b)}
			end   = []byte{byte(b + 1)}
		)
		if b == 255 {
			end = nil
		}
		log.Info("Compacting database", "range", fmt.Sprintf("%#X-%#X", start, end), "elapsed", common.PrettyDuration(time.Since(cstart)))
		if err := api.b.ChainDb().Compact(start, end); err != nil {
			log.Error("Database compaction failed", "err", err)
			return err
		}
	}
	return nil
}

// SetHead rewinds the head of the blockchain to a previous block.
func (api *DebugAPI) SetHead(number hexutil.Uint64) {
	api.b.SetHead(uint64(number))
}

func (api *DebugAPI) ChainConfig() *params.ChainConfig {
	return api.b.ChainConfig()
}

// NetAPI offers network related RPC methods
type NetAPI struct {
	net            *p2p.Server
	networkVersion uint64
}

// NewNetAPI creates a new net API instance.
func NewNetAPI(net *p2p.Server, networkVersion uint64) *NetAPI {
	return &NetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (api *NetAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers
func (api *NetAPI) PeerCount() hexutil.Uint {
	return hexutil.Uint(api.net.PeerCount())
}

// Version returns the current ethereum protocol version.
func (api *NetAPI) Version() string {
	return fmt.Sprintf("%d", api.networkVersion)
}

// checkTxFee is an internal function used to check whether the fee of
// the given transaction is _reasonable_(under the cap).
func checkTxFee(gasPrice *big.Int, gas uint64, cap float64) error {
	// Short circuit if there is no cap for transaction fee at all.
	if cap == 0 {
		return nil
	}
	feeEth := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gas))), new(big.Float).SetInt(big.NewInt(params.Ether)))
	feeFloat, _ := feeEth.Float64()
	if feeFloat > cap {
		return fmt.Errorf("tx fee (%.2f ether) exceeds the configured cap (%.2f ether)", feeFloat, cap)
	}
	return nil
}

// CheckTxFee exports a helper function used to check whether the fee is reasonable
func CheckTxFee(gasPrice *big.Int, gas uint64, cap float64) error {
	return checkTxFee(gasPrice, gas, cap)
}
