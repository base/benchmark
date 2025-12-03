package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rpc"
)

// StateOracle defines the high-level API used to retrieve L2 state data pre-images
// The returned data is always the preimage of the requested hash.
type StateOracle interface {
	// NodeByHash retrieves the merkle-patricia trie node pre-image for a given hash.
	// Trie nodes may be from the world state trie or any account storage trie.
	// Contract code is not stored as part of the trie and must be retrieved via CodeByHash
	NodeByHash(nodeHash common.Hash, chainID eth.ChainID) []byte

	// CodeByHash retrieves the contract code pre-image for a given hash.
	// codeHash should be retrieved from the world state account for a contract.
	CodeByHash(codeHash common.Hash, chainID eth.ChainID) []byte
}

type preimageOracle struct {
	db    ethdb.KeyValueStore
	codes map[common.Hash][]byte
	nodes map[common.Hash][]byte
}

func newPreimageOracle(db ethdb.KeyValueStore, codes map[common.Hash][]byte, nodes map[common.Hash][]byte) *preimageOracle {
	return &preimageOracle{
		db:    db,
		codes: codes,
		nodes: nodes,
	}
}

func (o *preimageOracle) CodeByHash(codeHash common.Hash, chainID eth.ChainID) []byte {
	return o.codes[codeHash]
}

func (o *preimageOracle) NodeByHash(nodeHash common.Hash, chainID eth.ChainID) []byte {
	return o.nodes[nodeHash]
}

var _ StateOracle = (*preimageOracle)(nil)

type gethPreimageOracle struct {
	db     ethdb.KeyValueStore
	client *ethclient.Client
}

func newGethPreimageOracle(db ethdb.KeyValueStore, client *ethclient.Client) *gethPreimageOracle {
	return &gethPreimageOracle{
		db:     db,
		client: client,
	}
}

const maxDbGetRetries = 5

// debugDbGet calls the debug_dbGet RPC method with retry logic.
// Retries up to maxDbGetRetries times with exponential backoff.
func debugDbGet(client *rpc.Client, key string) (hexutil.Bytes, error) {
	var result hexutil.Bytes
	var lastErr error

	for attempt := 0; attempt < maxDbGetRetries; attempt++ {
		err := client.CallContext(context.Background(), &result, "debug_dbGet", key)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't sleep after the last attempt
		if attempt < maxDbGetRetries-1 {
			backoff := time.Duration(1<<attempt) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("debug_dbGet failed after %d retries: %w", maxDbGetRetries, lastErr)
}

func (o *gethPreimageOracle) CodeByHash(codeHash common.Hash, chainID eth.ChainID) []byte {
	key := hexutil.Bytes(append(append(make([]byte, 0), rawdb.CodePrefix...), codeHash[:]...)).String()
	result, err := debugDbGet(o.client.Client(), key)
	if err != nil {
		panic(err)
	}
	return result
}

func (o *gethPreimageOracle) NodeByHash(nodeHash common.Hash, chainID eth.ChainID) []byte {
	result, err := debugDbGet(o.client.Client(), nodeHash.Hex())
	if err != nil {
		panic(err)
	}
	return result
}

type OracleKeyValueStore struct {
	db      ethdb.KeyValueStore
	oracle  StateOracle
	chainID eth.ChainID
}

func NewOracleBackedDB(kv ethdb.KeyValueStore, oracle StateOracle, chainID eth.ChainID) *OracleKeyValueStore {
	return &OracleKeyValueStore{
		db:      kv,
		oracle:  oracle,
		chainID: chainID,
	}
}

var codePrefixedKeyLength = common.HashLength + len(rawdb.CodePrefix)
var ErrInvalidKeyLength = errors.New("pre-images must be identified by 32-byte hash keys")

func (o *OracleKeyValueStore) Get(key []byte) ([]byte, error) {
	has, err := o.db.Has(key)
	if err != nil {
		return nil, fmt.Errorf("checking in-memory db: %w", err)
	}
	if has {
		return o.db.Get(key)
	}

	if len(key) == codePrefixedKeyLength && bytes.HasPrefix(key, rawdb.CodePrefix) {
		key = key[len(rawdb.CodePrefix):]
		return o.oracle.CodeByHash(*(*[common.HashLength]byte)(key), o.chainID), nil
	}
	if len(key) != common.HashLength {
		return nil, ErrInvalidKeyLength
	}
	return o.oracle.NodeByHash(*(*[common.HashLength]byte)(key), o.chainID), nil
}

func (o *OracleKeyValueStore) NewBatch() ethdb.Batch {
	return o.db.NewBatch()
}

func (o *OracleKeyValueStore) NewBatchWithSize(size int) ethdb.Batch {
	return o.db.NewBatchWithSize(size)
}

func (o *OracleKeyValueStore) Put(key []byte, value []byte) error {
	return o.db.Put(key, value)
}

func (o *OracleKeyValueStore) Close() error {
	return nil
}

// Remaining methods are unused when accessing the state for block processing so leaving unimplemented.

func (o *OracleKeyValueStore) Has(key []byte) (bool, error) {
	panic("not supported")
}

func (o *OracleKeyValueStore) Delete(key []byte) error {
	panic("not supported")
}

func (o *OracleKeyValueStore) DeleteRange(start, end []byte) error {
	panic("not supported")
}

func (o *OracleKeyValueStore) Stat() (string, error) {
	panic("not supported")
}

func (o *OracleKeyValueStore) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	panic("not supported")
}

func (o *OracleKeyValueStore) Compact(start []byte, limit []byte) error {
	panic("not supported")
}

func (o *OracleKeyValueStore) SyncKeyValue() error {
	return nil
}

var _ ethdb.KeyValueStore = (*OracleKeyValueStore)(nil)
