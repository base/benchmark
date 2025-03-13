package network

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/solabi"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type FakeConsensusClient struct {
	log        log.Logger
	client     *ethclient.Client
	authClient client.RPC

	headBlockHash common.Hash
	lastTimestamp uint64

	currentPayloadID *engine.PayloadID
}

func NewFakeConsensusClient(log log.Logger, client *ethclient.Client, authClient client.RPC, genesisHash common.Hash, genesisTimestamp uint64) *FakeConsensusClient {
	return &FakeConsensusClient{
		log:              log,
		client:           client,
		authClient:       authClient,
		headBlockHash:    genesisHash,
		lastTimestamp:    genesisTimestamp,
		currentPayloadID: nil,
	}
}

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

func (f *FakeConsensusClient) updateForkChoice(ctx context.Context) (*eth.PayloadID, error) {
	fcu := engine.ForkchoiceStateV1{
		HeadBlockHash:      f.headBlockHash,
		SafeBlockHash:      f.headBlockHash,
		FinalizedBlockHash: f.headBlockHash,
	}

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

	payloadAttrs := eth.PayloadAttributes{
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

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var resp engine.ForkChoiceResponse
	err = f.authClient.CallContext(ctx, &resp, "engine_forkchoiceUpdatedV3", fcu, payloadAttrs)

	if err != nil {
		return nil, errors.Wrap(err, "failed to propose block")
	}

	f.lastTimestamp = timestamp
	return resp.PayloadID, nil
}

func (f *FakeConsensusClient) getBuiltPayload(ctx context.Context, payloadID engine.PayloadID) (*engine.ExecutableData, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var payloadResp engine.ExecutionPayloadEnvelope
	err := f.authClient.CallContext(ctx, &payloadResp, "engine_getPayloadV4", payloadID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get payload")
	}

	f.log.Debug("Built payload", "parent_hash", payloadResp.ExecutionPayload.ParentHash, "stateRoot", payloadResp.ExecutionPayload.StateRoot)

	// get block
	ctx, cancel = context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	receipts, err := f.client.BlockReceipts(ctx, rpc.BlockNumberOrHash{BlockHash: &payloadResp.ExecutionPayload.ParentHash})
	if err != nil {
		f.log.Error("Failed to get receipts", "err", err)
	}

	// print receipts
	for i, receipt := range receipts {
		f.log.Debug("Parent receipts", "index", i, "status", receipt.Status, "gasUsed", receipt.GasUsed)
	}

	return payloadResp.ExecutionPayload, nil
}

func (f *FakeConsensusClient) newPayload(ctx context.Context, params *engine.ExecutableData) error {
	params.WithdrawalsRoot = &common.Hash{}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var resp engine.ForkChoiceResponse
	err := f.authClient.CallContext(ctx, &resp, "engine_newPayloadV4", params, []common.Hash{}, common.Hash{}, []common.Hash{})

	if err != nil {
		fmt.Printf("%#v\n", err)
		return errors.Wrap(err, "newPayload call failed")
	}

	return nil
}

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

	// wait 2 seconds
	time.Sleep(2000 * time.Millisecond)

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

func (f *FakeConsensusClient) Start(ctx context.Context) error {
	// min block time
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := f.Propose(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func (f *FakeConsensusClient) Stop() {

}
