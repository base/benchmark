package consensus

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/solabi"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

// SequencerConsensusClient is a fake consensus client that generates blocks on a timer.
type SequencerConsensusClient struct {
	*BaseConsensusClient
	mempool mempool.FakeMempool
}

// NewSequencerConsensusClient creates a new consensus client using the given genesis hash and timestamp.
func NewSequencerConsensusClient(log log.Logger, client *ethclient.Client, authClient client.RPC, mempool mempool.FakeMempool, genesis *core.Genesis, options ConsensusClientOptions) *SequencerConsensusClient {
	base := NewBaseConsensusClient(log, client, authClient, genesis, options)
	return &SequencerConsensusClient{
		BaseConsensusClient: base,
		mempool:             mempool,
	}
}

// marshalBinaryWithSignature creates the call data for an L1Info transaction.
func marshalBinaryWithSignature(info *derive.L1BlockInfo, signature []byte) ([]byte, error) {
	w := bytes.NewBuffer(make([]byte, 0, derive.L1InfoEcotoneLen))
	if err := solabi.WriteSignature(w, signature); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.BigEndian, info.BaseFeeScalar); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.BigEndian, info.BlobBaseFeeScalar); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.BigEndian, info.SequenceNumber); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.BigEndian, info.Time); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.BigEndian, info.Number); err != nil {
		return nil, err
	}
	if err := solabi.WriteUint256(w, info.BaseFee); err != nil {
		return nil, err
	}
	blobBasefee := info.BlobBaseFee
	if blobBasefee == nil {
		blobBasefee = big.NewInt(1) // set to 1, to match the min blob basefee as defined in EIP-4844
	}
	if err := solabi.WriteUint256(w, blobBasefee); err != nil {
		return nil, err
	}
	if err := solabi.WriteHash(w, info.BlockHash); err != nil {
		return nil, err
	}
	// ABI encoding will perform the left-padding with zeroes to 32 bytes, matching the "batcherHash" SystemConfig format and version 0 byte.
	if err := solabi.WriteAddress(w, info.BatcherAddr); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (f *SequencerConsensusClient) generatePayloadAttributes() (*eth.PayloadAttributes, error) {
	gasLimit := eth.Uint64Quantity(f.options.GasLimit)

	var b8 eth.Bytes8
	copy(b8[:], eip1559.EncodeHolocene1559Params(50, 1))

	timestamp := max(f.lastTimestamp+1, uint64(time.Now().Unix()))

	l1BlockInfo := &derive.L1BlockInfo{
		Number:         1,
		Time:           f.lastTimestamp,
		BaseFee:        big.NewInt(1),
		BlockHash:      common.Hash{},
		SequenceNumber: 0,
		BatcherAddr:    common.Address{},
	}

	source := derive.L1InfoDepositSource{
		L1BlockHash: common.Hash{},
		SeqNumber:   0,
	}

	data, err := marshalBinaryWithSignature(l1BlockInfo, derive.L1InfoFuncEcotoneBytes4)
	if err != nil {
		return nil, err
	}

	// Set a very large gas limit with `IsSystemTransaction` to ensure
	// that the L1 Attributes Transaction does not run out of gas.
	out := &types.DepositTx{
		SourceHash:          source.SourceHash(),
		From:                derive.L1InfoDepositerAddress,
		To:                  &derive.L1BlockAddress,
		Mint:                nil,
		Value:               big.NewInt(0),
		Gas:                 100_000,
		IsSystemTransaction: true,
		Data:                data,
	}
	l1Tx := types.NewTx(out)
	opaqueL1Tx, err := l1Tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to encode L1 info tx: %w", err)
	}

	payloadAttrs := &eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(timestamp),
		PrevRandao:            eth.Bytes32{},
		SuggestedFeeRecipient: common.Address{'C'},
		Withdrawals:           &types.Withdrawals{},
		Transactions:          []hexutil.Bytes{opaqueL1Tx},
		GasLimit:              &gasLimit,
		ParentBeaconBlockRoot: &common.Hash{},
		NoTxPool:              false,
		EIP1559Params:         &b8,
	}

	return payloadAttrs, nil
}

// Propose starts block generation, waits BlockTime, and generates a block.
func (f *SequencerConsensusClient) Propose(ctx context.Context, blockMetrics *metrics.BlockMetrics) (*engine.ExecutableData, error) {
	startTime := time.Now()

	transactionsToInclude := f.mempool.NextBlock()
	sendCallsPerBatch := 100
	batches := (len(transactionsToInclude) + sendCallsPerBatch - 1) / sendCallsPerBatch

	f.log.Info("Sending transactions", "num_transactions", len(transactionsToInclude), "num_batches", batches)

	for i := 0; i < batches; i++ {
		batch := transactionsToInclude[i*sendCallsPerBatch : min((i+1)*sendCallsPerBatch, len(transactionsToInclude))]
		results := make([]interface{}, len(batch))

		batchCall := make([]rpc.BatchElem, len(batch))
		for j, tx := range batch {
			batchCall[j] = rpc.BatchElem{
				Method: "eth_sendRawTransaction",
				Args:   []interface{}{hexutil.Encode(tx)},
				Result: &results[j],
			}
		}

		err := f.client.Client().BatchCallContext(ctx, batchCall)
		if err != nil {
			return nil, errors.Wrap(err, "failed to send transactions")
		}

		for _, tx := range batchCall {
			if tx.Error != nil {
				return nil, errors.Wrap(tx.Error, "failed to send transaction")
			}
		}
	}

	duration := time.Since(startTime)
	f.log.Info("Sent transactions", "duration", duration, "num_txs", len(transactionsToInclude))
	blockMetrics.AddExecutionMetric(metrics.SendTxsLatencyMetric, duration)
	startTime = time.Now()

	f.log.Info("Starting block building")

	payloadAttrs, err := f.generatePayloadAttributes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate payload attributes")
	}

	payloadID, err := f.updateForkChoice(ctx, payloadAttrs)
	if err != nil {
		return nil, err
	}

	duration = time.Since(startTime)
	blockMetrics.AddExecutionMetric(metrics.UpdateForkChoiceLatencyMetric, duration)

	f.currentPayloadID = payloadID

	// wait block time
	time.Sleep(f.options.BlockTime)

	startTime = time.Now()

	f.log.Info("Fetching built payload")

	payload, err := f.getBuiltPayload(ctx, *f.currentPayloadID)
	if err != nil {
		return nil, err
	}
	f.headBlockHash = payload.BlockHash
	f.headBlockNumber = payload.Number
	f.lastTimestamp = payload.Timestamp

	duration = time.Since(startTime)
	blockMetrics.AddExecutionMetric(metrics.GetPayloadLatencyMetric, duration)
	f.log.Info("Fetched built payload", "duration", duration, "txs", len(payload.Transactions), "BaseFeePerGas", payload.BaseFeePerGas)

	// get gas usage
	gasPerBlock := payload.GasUsed
	gasPerSecond := float64(gasPerBlock) / f.options.BlockTime.Seconds()
	blockMetrics.AddExecutionMetric(metrics.GasPerBlockMetric, float64(gasPerBlock))
	blockMetrics.AddExecutionMetric(metrics.GasPerSecondMetric, gasPerSecond)

	// get transactions per block
	transactionsPerBlock := len(payload.Transactions)
	blockMetrics.AddExecutionMetric(metrics.TransactionsPerBlockMetric, transactionsPerBlock)

	err = f.newPayload(ctx, payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}
