package mempool

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// ReplayMempool fetches transactions from an external node and replays them.
// It iterates through blocks from a source node and provides transactions
// block-by-block for the benchmark to replay.
type ReplayMempool struct {
	log    log.Logger
	client *ethclient.Client

	lock sync.Mutex

	// startBlock is the first block to fetch transactions from
	startBlock uint64

	// currentBlock tracks which block we're fetching next
	currentBlock uint64

	// chainID for transaction signing validation
	chainID *big.Int

	// addressNonce tracks the latest nonce for each address
	addressNonce map[common.Address]uint64
}

// NewReplayMempool creates a new ReplayMempool that fetches transactions
// from the given RPC endpoint starting from the specified block.
func NewReplayMempool(log log.Logger, rpcURL string, startBlock uint64, chainID *big.Int) (*ReplayMempool, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	return &ReplayMempool{
		log:          log,
		client:       client,
		startBlock:   startBlock,
		currentBlock: startBlock,
		chainID:      chainID,
		addressNonce: make(map[common.Address]uint64),
	}, nil
}

// AddTransactions is a no-op for ReplayMempool since transactions come from the source node.
func (m *ReplayMempool) AddTransactions(_ []*types.Transaction) {
	// No-op: transactions are fetched from the source node, not added manually
}

// NextBlock fetches the next block from the source node and returns its transactions.
// Returns (mempoolTxs, sequencerTxs) where:
// - mempoolTxs: regular transactions to be sent via eth_sendRawTransaction
// - sequencerTxs: deposit transactions to be included in payload attributes
func (m *ReplayMempool) NextBlock() ([][]byte, [][]byte) {
	m.lock.Lock()
	defer m.lock.Unlock()

	ctx := context.Background()

	block, err := m.client.BlockByNumber(ctx, big.NewInt(int64(m.currentBlock)))
	if err != nil {
		m.log.Warn("Failed to fetch block", "block", m.currentBlock, "error", err)
		return nil, nil
	}

	m.log.Info("Fetched block for replay",
		"block", m.currentBlock,
		"txs", len(block.Transactions()),
		"gas_used", block.GasUsed(),
	)

	m.currentBlock++

	mempoolTxs := make([][]byte, 0)
	sequencerTxs := make([][]byte, 0)

	for _, tx := range block.Transactions() {
		// Track nonces for GetTransactionCount
		from, err := types.Sender(types.NewIsthmusSigner(m.chainID), tx)
		if err != nil {
			// Try with London signer for older transactions
			from, err = types.Sender(types.NewLondonSigner(m.chainID), tx)
			if err != nil {
				m.log.Warn("Failed to get sender", "tx", tx.Hash(), "error", err)
				continue
			}
		}
		m.addressNonce[from] = tx.Nonce()

		txBytes, err := tx.MarshalBinary()
		if err != nil {
			m.log.Warn("Failed to marshal transaction", "tx", tx.Hash(), "error", err)
			continue
		}

		// Deposit transactions go to sequencer, others go to mempool
		if tx.Type() == types.DepositTxType {
			sequencerTxs = append(sequencerTxs, txBytes)
		} else {
			mempoolTxs = append(mempoolTxs, txBytes)
		}
	}

	return mempoolTxs, sequencerTxs
}

// GetTransactionCount returns the latest nonce for an address.
func (m *ReplayMempool) GetTransactionCount(address common.Address) uint64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.addressNonce[address]
}

// CurrentBlock returns the current block number being replayed.
func (m *ReplayMempool) CurrentBlock() uint64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.currentBlock
}

// Close closes the underlying RPC client connection.
func (m *ReplayMempool) Close() {
	m.client.Close()
}

var _ FakeMempool = &ReplayMempool{}

