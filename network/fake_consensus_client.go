package network

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

type FakeConsensusClient struct {
	client     *ethclient.Client
	authClient client.RPC

	headBlockHash common.Hash
	lastTimestamp uint64

	currentPayloadID *engine.PayloadID
}

func NewFakeConsensusClient(client *ethclient.Client, authClient client.RPC, genesisHash common.Hash, genesisTimestamp uint64) *FakeConsensusClient {
	return &FakeConsensusClient{
		client:           client,
		authClient:       authClient,
		headBlockHash:    genesisHash,
		lastTimestamp:    genesisTimestamp,
		currentPayloadID: nil,
	}
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

	payloadAttrs := eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(timestamp),
		PrevRandao:            eth.Bytes32{},
		SuggestedFeeRecipient: common.Address{'C'},
		Withdrawals:           &types.Withdrawals{},
		Transactions:          nil,
		GasLimit:              &gasLimit,
		ParentBeaconBlockRoot: &common.Hash{},
		NoTxPool:              false,
		EIP1559Params:         &b8,
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var resp engine.ForkChoiceResponse
	err := f.authClient.CallContext(ctx, &resp, "engine_forkchoiceUpdatedV3", fcu, payloadAttrs)

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

	return payloadResp.ExecutionPayload, nil
}

func (f *FakeConsensusClient) Propose(ctx context.Context) error {
	payloadID, err := f.updateForkChoice(ctx)
	if err != nil {
		return err
	}

	f.currentPayloadID = payloadID

	// wait 2 seconds
	time.Sleep(2000 * time.Millisecond)

	payload, err := f.getBuiltPayload(ctx, *f.currentPayloadID)
	if err != nil {
		return err
	}
	f.headBlockHash = payload.BlockHash

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
