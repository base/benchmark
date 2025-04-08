package consensus

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/base/base-bench/runner/metrics"
	"github.com/base/base-bench/runner/network/mempool"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
func NewSequencerConsensusClient(log log.Logger, client *ethclient.Client, authClient client.RPC, mempool mempool.FakeMempool, genesis *core.Genesis, metricsCollector metrics.MetricsCollector, options ConsensusClientOptions) *SequencerConsensusClient {
	base := NewBaseConsensusClient(log, client, authClient, genesis, metricsCollector, options)
	return &SequencerConsensusClient{
		BaseConsensusClient: base,
		mempool:             mempool,
	}
}

func (f *SequencerConsensusClient) generatePayloadAttributes() (*eth.PayloadAttributes, error) {
	gasLimit := eth.Uint64Quantity(40e9)

	var b8 eth.Bytes8
	copy(b8[:], eip1559.EncodeHolocene1559Params(50, 10))

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
		Gas:                 150_000_000,
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

func (f *SequencerConsensusClient) updateForkChoice(ctx context.Context, payloadAttrs *eth.PayloadAttributes) (*eth.PayloadID, error) {
	fcu := engine.ForkchoiceStateV1{
		HeadBlockHash:      f.headBlockHash,
		SafeBlockHash:      f.headBlockHash,
		FinalizedBlockHash: f.headBlockHash,
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var resp engine.ForkChoiceResponse
	err := f.authClient.CallContext(ctx, &resp, "engine_forkchoiceUpdatedV3", fcu, payloadAttrs)

	if err != nil {
		return nil, errors.Wrap(err, "failed to propose block")
	}

	if resp.PayloadID == nil {
		return nil, fmt.Errorf("failed to propose block, payload status: %#v", resp.PayloadStatus)
	}

	f.lastTimestamp = uint64(payloadAttrs.Timestamp)
	return resp.PayloadID, nil
}

// Propose starts block generation, waits BlockTime, and generates a block.
func (f *SequencerConsensusClient) Propose(ctx context.Context) (*engine.ExecutableData, error) {
	payloadAttrs, err := f.generatePayloadAttributes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate payload attributes")
	}

	f.log.Info("Starting block building")

	payloadID, err := f.updateForkChoice(ctx, payloadAttrs)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()

	transactionsToInclude := f.mempool.NextBlock()
	sendCallsPerBatch := 100
	batches := len(transactionsToInclude) / sendCallsPerBatch

	f.log.Info("Sending transactions", "num_transactions", len(transactionsToInclude), "num_batches", batches)

	for i := 0; i < batches; i++ {
		batch := transactionsToInclude[i*sendCallsPerBatch : (i+1)*sendCallsPerBatch]
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

	f.currentPayloadID = payloadID

	// wait block time
	time.Sleep(f.options.BlockTime - duration)

	startTime = time.Now()

	f.log.Info("Fetching built payload")

	payload, err := f.getBuiltPayload(ctx, *f.currentPayloadID)
	if err != nil {
		return nil, err
	}
	f.headBlockHash = payload.BlockHash

	duration = time.Since(startTime)
	f.log.Info("Fetched built payload", "duration", duration)

	err = f.newPayload(ctx, payload)
	if err != nil {
		return nil, err
	}

	// Collect metrics after each block
	f.collectMetrics(ctx)
	return payload, nil
}
