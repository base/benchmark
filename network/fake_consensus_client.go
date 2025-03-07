package network

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

type FakeConsensusClient struct {
	client *ethclient.Client

	headBlockHash common.Hash
}

func NewFakeConsensusClient(client *ethclient.Client, genesisHash common.Hash) *FakeConsensusClient {
	return &FakeConsensusClient{
		client:        client,
		headBlockHash: genesisHash,
	}
}

func (f *FakeConsensusClient) Propose(ctx context.Context) error {
	fcu := engine.ForkchoiceStateV1{
		HeadBlockHash: f.headBlockHash,
	}

	payloadAttrs := engine.PayloadAttributes{
		Timestamp:             uint64(time.Now().Unix()),
		Random:                common.Hash{},
		SuggestedFeeRecipient: common.Address{},
		Withdrawals:           nil,
		Transactions:          [][]byte{},
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var resp engine.ForkChoiceResponse
	err := f.client.Client().CallContext(ctx, &resp, "engine_forkChoiceUpdatedV3", fcu, payloadAttrs)

	if err != nil {
		return errors.Wrap(err, "failed to propose block")
	}

	// wait 2 seconds
	time.Sleep(2 * time.Second)

	ctx, cancel = context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var payloadResp engine.ExecutionPayloadEnvelope
	err = f.client.Client().CallContext(ctx, &payloadResp, "engine_getPayloadV4", resp)
	if err != nil {
		return errors.Wrap(err, "failed to get payload")
	}

	fmt.Printf("success: %#v", payloadResp)
	return nil
}

func (f *FakeConsensusClient) Start(ctx context.Context) error {
	return f.Propose(ctx)
}

func (f *FakeConsensusClient) Stop() {

}
