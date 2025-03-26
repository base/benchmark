package mempool

import "sync"

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

	// transactionsByBlock[block][txIndex] represents each transaction in the block.
	transactionsByBlock [][][]byte

	currentBlock [][]byte
	currentGas   uint64
	gasPerBlock  uint64
}

func NewStaticWorkloadMempool(gasPerBlock uint64) *StaticWorkloadMempool {
	return &StaticWorkloadMempool{gasPerBlock: gasPerBlock}
}

func (m *StaticWorkloadMempool) AddTransaction(transaction []byte, gasUsed uint64) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if (m.currentGas + gasUsed) > m.gasPerBlock {
		m.transactionsByBlock = append(m.transactionsByBlock, m.currentBlock)
		m.currentBlock = nil
		m.currentGas = 0
	}
	m.currentBlock = append(m.currentBlock, transaction)
	m.currentGas += gasUsed
}

func (m *StaticWorkloadMempool) NextBlock() [][]byte {
	m.lock.Lock()
	defer m.lock.Unlock()
	if len(m.transactionsByBlock) == 0 {
		if len(m.currentBlock) > 0 {
			block := m.currentBlock
			m.currentBlock = nil
			m.currentGas = 0
			return block
		}
		return [][]byte{}
	}
	block := m.transactionsByBlock[0]
	m.transactionsByBlock = m.transactionsByBlock[1:]
	return block
}

var _ FakeMempool = &StaticWorkloadMempool{}
