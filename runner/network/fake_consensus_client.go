package network

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
	"github.com/pkg/errors"
)

// FakeConsensusClientOptions is an object for configuring a FakeConsensusClient.
type FakeConsensusClientOptions struct {
	// BlockTime is the time between FCU and GetPayload calls
	BlockTime time.Duration
}

// FakeConsensusClient is a fake consensus client that generates blocks on a timer.
type FakeConsensusClient struct {
	log        log.Logger
	client     *ethclient.Client
	authClient client.RPC
	options    FakeConsensusClientOptions
	mempool    mempool.FakeMempool

	headBlockHash common.Hash
	lastTimestamp uint64

	currentPayloadID *engine.PayloadID
	metricsCollector metrics.MetricsCollector
}

// NewFakeConsensusClient creates a new consensus client using the given genesis hash and timestamp.
func NewFakeConsensusClient(log log.Logger, client *ethclient.Client, authClient client.RPC, mempool mempool.FakeMempool, genesis *core.Genesis, metricsCollector metrics.MetricsCollector, options FakeConsensusClientOptions) *FakeConsensusClient {
	genesisHash := genesis.ToBlock().Hash()
	genesisTimestamp := genesis.Timestamp

	return &FakeConsensusClient{
		log:              log,
		client:           client,
		authClient:       authClient,
		headBlockHash:    genesisHash,
		lastTimestamp:    genesisTimestamp,
		options:          options,
		mempool:          mempool,
		currentPayloadID: nil,
		metricsCollector: metricsCollector,
	}
}

func (f *FakeConsensusClient) generatePayloadAttributes() (*eth.PayloadAttributes, error) {
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

	txBytes := f.mempool.NextBlock()
	hexBytes := make([]hexutil.Bytes, len(txBytes))
	for i, tx := range txBytes {
		hexBytes[i] = tx
	}

	f.log.Info("Generated payload attributes", "timestamp", timestamp, "num_txs", len(hexBytes))

	payloadAttrs := &eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(timestamp),
		PrevRandao:            eth.Bytes32{},
		SuggestedFeeRecipient: common.Address{'C'},
		Withdrawals:           &types.Withdrawals{},
		Transactions:          append([]hexutil.Bytes{opaqueL1Tx}, hexBytes...),
		GasLimit:              &gasLimit,
		ParentBeaconBlockRoot: &common.Hash{},
		NoTxPool:              true,
		EIP1559Params:         &b8,
	}

	return payloadAttrs, nil
}

func (f *FakeConsensusClient) updateForkChoice(ctx context.Context) (*eth.PayloadID, error) {
	fcu := engine.ForkchoiceStateV1{
		HeadBlockHash:      f.headBlockHash,
		SafeBlockHash:      f.headBlockHash,
		FinalizedBlockHash: f.headBlockHash,
	}

	payloadAttrs, err := f.generatePayloadAttributes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate payload attributes")
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var resp engine.ForkChoiceResponse
	err = f.authClient.CallContext(ctx, &resp, "engine_forkchoiceUpdatedV3", fcu, payloadAttrs)

	if err != nil {
		return nil, errors.Wrap(err, "failed to propose block")
	}

	f.lastTimestamp = uint64(payloadAttrs.Timestamp)
	return resp.PayloadID, nil
}

func (f *FakeConsensusClient) getBuiltPayload(ctx context.Context, payloadID engine.PayloadID) (*engine.ExecutableData, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var payloadResp engine.ExecutionPayloadEnvelope
	err := f.authClient.CallContext(ctx, &payloadResp, "engine_getPayloadV3", payloadID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get payload")
	}

	f.log.Debug("Built payload", "parent_hash", payloadResp.ExecutionPayload.ParentHash, "stateRoot", payloadResp.ExecutionPayload.StateRoot)

	return payloadResp.ExecutionPayload, nil
}

// BasicBlockType implements what chain config would usually implement.
type basicBlockType struct{}

// HasOptimismWithdrawalsRoot implements types.BlockType.
func (b basicBlockType) HasOptimismWithdrawalsRoot(blkTime uint64) bool {
	return false
}

// IsIsthmus implements types.BlockType.
func (b basicBlockType) IsIsthmus(blkTime uint64) bool {
	return false
}

var _ types.BlockType = basicBlockType{}

var (
	EmptyWithdrawalsRoot = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
)

// newPayload calls engine_newPayloadV3 with the given executable data, filling out necessary info.
func (f *FakeConsensusClient) newPayload(ctx context.Context, params *engine.ExecutableData) error {
	block, err := engine.ExecutableDataToBlockNoHash(*params, []common.Hash{}, &common.Hash{}, nil, basicBlockType{})
	if err != nil {
		return errors.Wrap(err, "failed to convert payload to block")
	}

	params.BlockHash = block.Hash()

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var resp engine.ForkChoiceResponse
	err = f.authClient.CallContext(ctx, &resp, "engine_newPayloadV3", params, []common.Hash{}, common.Hash{})

	if err != nil {
		return errors.Wrap(err, "newPayload call failed")
	}

	return nil
}

// Propose starts block generation, waits BlockTime, and generates a block.
func (f *FakeConsensusClient) Propose(ctx context.Context) error {
	payloadID, err := f.updateForkChoice(ctx)
	if err != nil {
		return err
	}

	f.currentPayloadID = payloadID

	if f.currentPayloadID == nil {
		log.Warn("No current payload ID")
		return nil
	}

	// wait block time
	time.Sleep(f.options.BlockTime)

	payload, err := f.getBuiltPayload(ctx, *f.currentPayloadID)
	if err != nil {
		return err
	}
	f.headBlockHash = payload.BlockHash

	err = f.newPayload(ctx, payload)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the fake consensus client.
func (f *FakeConsensusClient) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := f.Propose(ctx)
			if err != nil {
				return err
			}

			// Collect metrics after each block
			if err := f.metricsCollector.Collect(ctx); err != nil {
				f.log.Error("Failed to collect metrics", "error", err)
				continue
			}
		}
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
