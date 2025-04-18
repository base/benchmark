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
	// NextBlock returns the next block of transactions to be included in the chain.
	NextBlock() [][]byte
}

type StaticWorkloadMempool struct {
	// needs to be thread safe to share between workers (could be converted to channel)
	lock sync.Mutex
	log  log.Logger

	addressNonce map[common.Address]uint64

	currentBlock [][]byte
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

		m.currentBlock = append(m.currentBlock, bytes)
	}
}

// returns nonce of latest transaction. This will be incremented by the transaction generators.
func (m *StaticWorkloadMempool) GetTransactionCount(address common.Address) uint64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.addressNonce[address]
}

func (m *StaticWorkloadMempool) NextBlock() [][]byte {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.currentBlock) == 0 {
		return [][]byte{}
	}

	block := m.currentBlock
	m.currentBlock = nil
	return block
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
