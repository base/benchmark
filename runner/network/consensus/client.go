package consensus

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// ConsensusClientOptions is an object for configuring a ConsensusClient.
type ConsensusClientOptions struct {
	// BlockTime is the time between FCU and GetPayload calls
	BlockTime time.Duration
	// GasLimit is the gas limit for the payload
	GasLimit uint64
	// GasLimitSetup is the gas limit for the setup payload
	GasLimitSetup uint64
}

// BaseConsensusClient contains common functionality shared between different consensus client implementations.
type BaseConsensusClient struct {
	log        log.Logger
	client     *ethclient.Client
	authClient client.RPC
	options    ConsensusClientOptions

	headBlockHash   common.Hash
	headBlockNumber uint64

	currentPayloadID *engine.PayloadID
}

// NewBaseConsensusClient creates a new base consensus client.
func NewBaseConsensusClient(log log.Logger, client *ethclient.Client, authClient client.RPC, options ConsensusClientOptions, headBlockHash common.Hash, headBlockNumber uint64) *BaseConsensusClient {
	return &BaseConsensusClient{
		log:              log,
		client:           client,
		authClient:       authClient,
		headBlockHash:    headBlockHash,
		headBlockNumber:  headBlockNumber,
		options:          options,
		currentPayloadID: nil,
	}
}

func (f *BaseConsensusClient) updateForkChoice(ctx context.Context, payloadAttrs *eth.PayloadAttributes) (*eth.PayloadID, error) {
	fcu := engine.ForkchoiceStateV1{
		HeadBlockHash:      f.headBlockHash,
		SafeBlockHash:      f.headBlockHash,
		FinalizedBlockHash: f.headBlockHash,
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var resp engine.ForkChoiceResponse
	err := f.authClient.CallContext(ctx, &resp, "engine_forkchoiceUpdatedV3", fcu, payloadAttrs)

	if err != nil {
		return nil, errors.Wrap(err, "failed to propose block")
	}

	return resp.PayloadID, nil
}

// getBuiltPayload retrieves the built payload for the given payload ID.
func (b *BaseConsensusClient) getBuiltPayload(ctx context.Context, payloadID engine.PayloadID) (*engine.ExecutableData, error) {
	ctx, cancel := context.WithTimeout(ctx, 240*time.Second)
	defer cancel()
	var payloadResp engine.ExecutionPayloadEnvelope
	err := b.authClient.CallContext(ctx, &payloadResp, "engine_getPayloadV4", payloadID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get payload")
	}

	b.log.Debug("Built payload", "parent_hash", payloadResp.ExecutionPayload.ParentHash, "stateRoot", payloadResp.ExecutionPayload.StateRoot)

	return payloadResp.ExecutionPayload, nil
}

// newPayload calls engine_newPayloadV4 with the given executable data.
func (b *BaseConsensusClient) newPayload(ctx context.Context, params *engine.ExecutableData, beaconRoot common.Hash) error {

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var resp engine.ForkChoiceResponse
	err := b.authClient.CallContext(ctx, &resp, "engine_newPayloadV4", params, []common.Hash{}, beaconRoot, []common.Hash{})

	if err != nil {
		return errors.Wrap(err, "newPayload call failed")
	}

	return nil
}
