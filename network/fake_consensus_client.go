package network

import (
	"context"
	"fmt"
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

	headBlockHash    common.Hash
	genesisTimestamp uint64
}

func NewFakeConsensusClient(client *ethclient.Client, authClient client.RPC, genesisHash common.Hash, genesisTimestamp uint64) *FakeConsensusClient {
	return &FakeConsensusClient{
		client:           client,
		authClient:       authClient,
		headBlockHash:    genesisHash,
		genesisTimestamp: genesisTimestamp,
	}
}

func (f *FakeConsensusClient) Propose(ctx context.Context) error {
	fcu := engine.ForkchoiceStateV1{
		HeadBlockHash:      f.headBlockHash,
		SafeBlockHash:      f.headBlockHash,
		FinalizedBlockHash: f.headBlockHash,
	}

	gasLimit := eth.Uint64Quantity(40e9)

	var b8 eth.Bytes8
	copy(b8[:], eip1559.EncodeHolocene1559Params(50, 10))

	payloadAttrs := eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(f.genesisTimestamp + 2),
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
		fmt.Printf("%#+v\n", err)
		return errors.Wrap(err, "failed to propose block")
	}

	// wait 2 seconds
	time.Sleep(200 * time.Millisecond)

	fmt.Println(resp)

	ctx, cancel = context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var payloadResp engine.ExecutionPayloadEnvelope
	err = f.authClient.CallContext(ctx, &payloadResp, "engine_getPayloadV4", *resp.PayloadID)
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
