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
	"github.com/base/base-bench/runner/network/proofprogram/fakel1"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/solabi"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

// SequencerConsensusClient is a fake consensus client that generates blocks on a timer.
type SequencerConsensusClient struct {
	*BaseConsensusClient
	lastTimestamp uint64
	mempool       mempool.FakeMempool
	l1Chain       fakel1.L1Chain
	batcherAddr   common.Address
}

// NewSequencerConsensusClient creates a new consensus client using the given genesis hash and timestamp.
func NewSequencerConsensusClient(log log.Logger, client *ethclient.Client, authClient client.RPC, mempool mempool.FakeMempool, options ConsensusClientOptions, headBlockHash common.Hash, headBlockNumber uint64, l1Chain fakel1.L1Chain, batcherAddr common.Address) *SequencerConsensusClient {
	base := NewBaseConsensusClient(log, client, authClient, options, headBlockHash, headBlockNumber)
	return &SequencerConsensusClient{
		BaseConsensusClient: base,
		lastTimestamp:       uint64(time.Now().Unix()),
		mempool:             mempool,
		l1Chain:             l1Chain,
		batcherAddr:         batcherAddr,
	}
}

func (f *SequencerConsensusClient) Stop(ctx context.Context) error {
	f.log.Info("Stopping sequencer consensus client")

	// ensure fork choice is updated to the last block
	_, err := f.updateForkChoice(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

// marshalBinaryWithSignature creates the call data for an L1Info transaction.
func marshalBinaryWithSignature(info *derive.L1BlockInfo, signature []byte) ([]byte, error) {
	w := bytes.NewBuffer(make([]byte, 0, derive.L1InfoIsthmusLen))
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
	if err := binary.Write(w, binary.BigEndian, info.OperatorFeeScalar); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.BigEndian, info.OperatorFeeConstant); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (f *SequencerConsensusClient) generatePayloadAttributes(sequencerTxs [][]byte, isSetupPayload bool) (*eth.PayloadAttributes, *common.Hash, error) {
	gasLimit := eth.Uint64Quantity(f.options.GasLimit)
	if isSetupPayload {
		gasLimit = eth.Uint64Quantity(f.options.GasLimitSetup)
	}

	var b8 eth.Bytes8
	copy(b8[:], eip1559.EncodeHolocene1559Params(50, 1))

	timestamp := f.lastTimestamp + 1

	number := uint64(0)
	time := uint64(0)
	baseFee := big.NewInt(1)
	blockHash := common.Hash{}
	if f.l1Chain != nil {
		block, err := f.l1Chain.GetBlockByNumber(1)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get block by number: %w", err)
		}
		number = block.NumberU64()
		time = block.Time()
		baseFee = block.BaseFee()
		blockHash = block.Hash()
	}

	l1BlockInfo := &derive.L1BlockInfo{
		Number:              number,
		Time:                time,
		BaseFee:             baseFee,
		BlockHash:           blockHash,
		SequenceNumber:      f.headBlockNumber,
		BatcherAddr:         f.batcherAddr,
		BlobBaseFee:         big.NewInt(1),
		BaseFeeScalar:       1,
		BlobBaseFeeScalar:   1,
		OperatorFeeScalar:   0,
		OperatorFeeConstant: 0,
	}

	source := derive.L1InfoDepositSource{
		L1BlockHash: l1BlockInfo.BlockHash,
		SeqNumber:   l1BlockInfo.SequenceNumber,
	}

	data, err := marshalBinaryWithSignature(l1BlockInfo, derive.L1InfoFuncIsthmusBytes4)
	if err != nil {
		return nil, nil, err
	}

	// Set a very large gas limit with `IsSystemTransaction` to ensure
	// that the L1 Attributes Transaction does not run out of gas.
	out := &types.DepositTx{
		SourceHash:          source.SourceHash(),
		From:                derive.L1InfoDepositerAddress,
		To:                  &derive.L1BlockAddress,
		Mint:                nil,
		Value:               big.NewInt(0),
		Gas:                 1_000_000,
		IsSystemTransaction: false,
		Data:                data,
	}
	l1Tx := types.NewTx(out)
	opaqueL1Tx, err := l1Tx.MarshalBinary()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode L1 info tx: %w", err)
	}

	sequencerTxsHexBytes := make([]hexutil.Bytes, len(sequencerTxs)+1)
	sequencerTxsHexBytes[0] = hexutil.Bytes(opaqueL1Tx)
	for i, tx := range sequencerTxs {
		sequencerTxsHexBytes[i+1] = hexutil.Bytes(tx)
	}

	root := crypto.Keccak256Hash([]byte("fake-beacon-block-root"), big.NewInt(int64(1)).Bytes())

	payloadAttrs := &eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(timestamp),
		PrevRandao:            eth.Bytes32{},
		SuggestedFeeRecipient: common.HexToAddress("0x4200000000000000000000000000000000000011"),
		Withdrawals:           &types.Withdrawals{},
		Transactions:          sequencerTxsHexBytes,
		GasLimit:              &gasLimit,
		ParentBeaconBlockRoot: &root,
		NoTxPool:              false,
		EIP1559Params:         &b8,
	}

	return payloadAttrs, &root, nil
}

// Propose starts block generation, waits BlockTime, and generates a block.
func (f *SequencerConsensusClient) Propose(ctx context.Context, blockMetrics *metrics.BlockMetrics, isSetupPayload bool) (*engine.ExecutableData, error) {
	startTime := time.Now()

	sendTxs, sequencerTxs := f.mempool.NextBlock()

	sendCallsPerBatch := 100
	batches := (len(sendTxs) + sendCallsPerBatch - 1) / sendCallsPerBatch

	for i := 0; i < batches; i++ {
		batch := sendTxs[i*sendCallsPerBatch : min((i+1)*sendCallsPerBatch, len(sendTxs))]
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
				return nil, errors.Wrapf(tx.Error, "failed to send transaction %#v", tx.Args[0])
			}
		}
	}

	duration := time.Since(startTime)
	f.log.Info("Sent transactions", "duration", duration, "num_txs", len(sendTxs))
	blockMetrics.AddExecutionMetric(metrics.SendTxsLatencyMetric, duration)
	startBlockBuildingTime := time.Now()

	f.log.Info("Starting block building")

	payloadAttrs, beaconRoot, err := f.generatePayloadAttributes(sequencerTxs, isSetupPayload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate payload attributes")
	}

	startTime = time.Now()
	payloadID, err := f.updateForkChoice(ctx, payloadAttrs)
	if err != nil {
		return nil, err
	}

	if payloadID == nil {
		return nil, errors.New("failed to build block")
	}
	duration = time.Since(startTime)
	blockMetrics.AddExecutionMetric(metrics.UpdateForkChoiceLatencyMetric, duration)

	f.currentPayloadID = payloadID
	// wait block time
	time.Sleep(f.options.BlockTime)

	startTime = time.Now()

	payload, err := f.getBuiltPayload(ctx, *f.currentPayloadID)
	if err != nil {
		return nil, err
	}
	f.headBlockHash = payload.BlockHash
	f.headBlockNumber = payload.Number
	f.lastTimestamp = payload.Timestamp
	blockBuildingDuration := time.Since(startBlockBuildingTime)

	duration = time.Since(startTime)
	blockMetrics.AddExecutionMetric(metrics.GetPayloadLatencyMetric, duration)
	f.log.Info("Fetched built payload", "duration", duration, "txs", len(payload.Transactions), "number", payload.Number, "hash", payload.BlockHash.Hex())

	// get gas usage
	gasPerBlock := payload.GasUsed
	gasPerSecond := float64(gasPerBlock) / blockBuildingDuration.Seconds()
	blockMetrics.AddExecutionMetric(metrics.GasPerBlockMetric, float64(gasPerBlock))
	blockMetrics.AddExecutionMetric(metrics.GasPerSecondMetric, gasPerSecond)

	// get transactions per block
	transactionsPerBlock := len(payload.Transactions)
	blockMetrics.AddExecutionMetric(metrics.TransactionsPerBlockMetric, transactionsPerBlock)

	err = f.newPayload(ctx, payload, *beaconRoot)
	if err != nil {
		return nil, err
	}

	return payload, nil
}
