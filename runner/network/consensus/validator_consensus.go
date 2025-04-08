package consensus

import (
	"bytes"
	"context"
	"encoding/binary"
	"math/big"
	"time"

	"github.com/base/base-bench/runner/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/solabi"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// SyncingConsensusClient is a fake consensus client that generates blocks on a timer.
type SyncingConsensusClient struct {
	*BaseConsensusClient
}

// NewSyncingConsensusClient creates a new consensus client.
func NewSyncingConsensusClient(log log.Logger, client *ethclient.Client, authClient client.RPC, genesis *core.Genesis, metricsCollector metrics.MetricsCollector, options ConsensusClientOptions) *SyncingConsensusClient {
	base := NewBaseConsensusClient(log, client, authClient, genesis, metricsCollector, options)
	return &SyncingConsensusClient{
		BaseConsensusClient: base,
	}
}

// Propose starts block generation, waits BlockTime, and generates a block.
func (f *SyncingConsensusClient) Propose(ctx context.Context, payload *engine.ExecutableData) error {
	f.log.Info("Updating fork choice before validating payload", "payload_index", payload.Number)
	_, err := f.updateForkChoice(ctx)
	if err != nil {
		return err
	}

	f.log.Info("Validate payload", "payload_index", payload.Number)
	startTime := time.Now()
	err = f.newPayload(ctx, payload)
	if err != nil {
		return err
	}
	duration := time.Since(startTime)
	f.log.Info("Validated payload", "payload_index", payload.Number, "duration", duration)

	return nil
}

// Start starts the fake consensus client.
func (f *SyncingConsensusClient) Start(ctx context.Context, payloads []engine.ExecutableData) error {
	f.log.Info("Starting sync benchmark", "num_payloads", len(payloads))
	for i := 0; i < len(payloads); i++ {
		f.log.Info("Proposing payload", "payload_index", i)
		err := f.Propose(ctx, &payloads[i])
		if err != nil {
			return err
		}

		// Collect metrics after each block
		f.collectMetrics(ctx)
	}
	return nil
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
