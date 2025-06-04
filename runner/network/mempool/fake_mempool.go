package mempool

import (
	"encoding/hex"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// FakeMempool emulates what the mempool would generally do (organize transactions into blocks).
// This can be implemented as either a static workload of known gas usage, or a dynamic workload
// that is loaded from a file after being simulated first.
type FakeMempool interface {
	// AddTransactions adds transactions to the mempool (thread-safe).
	AddTransactions(transactions []*types.Transaction)

	// NextBlock returns the next block of transactions to be included in the chain.
	NextBlock() (sendTxs [][]byte, sequencerTxs [][]byte)
}

// StaticWorkloadMempool is a fake mempool that simulates a workload of transactions with no gas
// or dependency tracking.
type StaticWorkloadMempool struct {
	// needs to be thread safe to share between workers (could be converted to channel)
	lock sync.Mutex
	log  log.Logger

	addressNonce map[common.Address]uint64

	// normal block txs submitted through mempool
	currentBlockTxs [][]byte

	// sequencer txs included in payload attributes
	currentBlockSequencerTxs [][]byte
}

func NewStaticWorkloadMempool(log log.Logger) *StaticWorkloadMempool {

	return &StaticWorkloadMempool{
		log:          log,
		addressNonce: make(map[common.Address]uint64),
	}
}

func (m *StaticWorkloadMempool) AddTransactions(transactions []*types.Transaction) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, transaction := range transactions {
		from, err := types.Sender(types.NewIsthmusSigner(transaction.ChainId()), transaction)

		if err != nil {
			panic(err)
		}

		m.addressNonce[from] = transaction.Nonce()

		bytes, err := transaction.MarshalBinary()
		if err != nil {
			panic(err)
		}

		if transaction.Type() != types.DepositTxType {
			m.currentBlockTxs = append(m.currentBlockTxs, bytes)
		} else {
			m.currentBlockSequencerTxs = append(m.currentBlockSequencerTxs, bytes)
		}
	}
}

// returns nonce of latest transaction. This will be incremented by the transaction generators.
func (m *StaticWorkloadMempool) GetTransactionCount(address common.Address) uint64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.addressNonce[address]
}

func (m *StaticWorkloadMempool) NextBlock() ([][]byte, [][]byte) {
	m.lock.Lock()
	defer m.lock.Unlock()

	block := m.currentBlockTxs
	blockSequencerTxs := m.currentBlockSequencerTxs
	m.currentBlockTxs = nil
	m.currentBlockSequencerTxs = nil

	return block, blockSequencerTxs
}

var _ FakeMempool = &StaticWorkloadMempool{}

func (m *StaticWorkloadMempool) DebugTransaction(from *common.Address, tx *types.Transaction) {
	m.log.Debug("Transaction details",
		"from", from.Hex(),
		"nonce", tx.Nonce(),
		"to", tx.To().Hex(),
		"value", tx.Value().String(),
		"gas", tx.Gas(),
		"gasPrice", tx.GasPrice().String(),
		"data", hex.EncodeToString(tx.Data()),
	)

	v, r, s := tx.RawSignatureValues()
	m.log.Debug("Transaction signature",
		"V", v.String(),
		"R", r.String(),
		"S", s.String(),
	)
}
