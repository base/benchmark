package proofprogram

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"io"
	"math/big"

	"github.com/base/base-bench/runner/network/proofprogram/fakel1"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derive_params "github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
)

// Batcher handles the creation and submission of L2 batches to L1
type Batcher struct {
	rollupCfg   *rollup.Config
	batcherKey  *ecdsa.PrivateKey
	batcherAddr common.Address
	chain       fakel1.L1Chain
	maxL1TxSize uint64
}

// NewBatcher creates a new batcher instance
func NewBatcher(rollupCfg *rollup.Config, batcherKey *ecdsa.PrivateKey, chain fakel1.L1Chain) *Batcher {
	return &Batcher{
		rollupCfg:   rollupCfg,
		batcherKey:  batcherKey,
		batcherAddr: crypto.PubkeyToAddress(batcherKey.PublicKey),
		chain:       chain,
		maxL1TxSize: 128 * 1024,
	}
}

// CreateSpanBatch creates a span batch from the given payloads
func (b *Batcher) CreateSpanBatches(payloads []engine.ExecutableData) ([][]byte, error) {
	target := batcher.MaxDataSize(1, b.maxL1TxSize)
	chainSpec := rollup.NewChainSpec(b.rollupCfg)
	ch, err := derive.NewSpanChannelOut(target, derive.Zlib, chainSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to create span channel: %w", err)
	}

	frames := make([][]byte, 0)

	for _, payload := range payloads {
		root := crypto.Keccak256Hash([]byte("fake-beacon-block-root"), big.NewInt(1).Bytes())

		block, err := engine.ExecutableDataToBlock(payload, []common.Hash{}, &root, [][]byte{}, benchtypes.IsthmusBlockType{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert payload to block")
		}

		// Try to add the block to the channel
		_, err = ch.AddBlock(b.rollupCfg, block)
		if err != nil {
			// If the channel is full, flush it first
			if err == derive.ErrCompressorFull {
				// Flush the current channel
				data := new(bytes.Buffer)
				data.WriteByte(derive_params.DerivationVersion0)

				if _, err = ch.OutputFrame(data, b.maxL1TxSize-1); err != nil && err != io.EOF {
					return nil, fmt.Errorf("failed to output frame: %w", err)
				}

				if err == io.EOF {
					break
				}

				frames = append(frames, data.Bytes())
				data.Reset()
				data.WriteByte(derive_params.DerivationVersion0)

				// Create a new channel for remaining blocks
				ch, err = derive.NewSpanChannelOut(target, derive.Zlib, chainSpec)
				if err != nil {
					return nil, fmt.Errorf("failed to create new span channel: %w", err)
				}

				// Try adding the block again to the new channel
				_, err = ch.AddBlock(b.rollupCfg, block)
				if err != nil {
					return nil, fmt.Errorf("failed to add block to new channel: %w", err)
				}
			} else {
				return nil, fmt.Errorf("failed to add block to channel: %w", err)
			}
		}
	}

	// Flush any remaining data in the channel
	err = ch.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close channel: %w", err)
	}

	if ch.ReadyBytes() > 0 {

		data := new(bytes.Buffer)
		data.WriteByte(derive_params.DerivationVersion0)

		if _, err = ch.OutputFrame(data, b.maxL1TxSize-1); err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to output frame: %w", err)
		}

		frames = append(frames, data.Bytes())
	}

	return frames, nil
}

// CreateAndSendBatch creates a batch and prepares it for sending to L1
func (b *Batcher) CreateAndSendBatch(payloads []engine.ExecutableData, parentHash common.Hash) error {
	frames, err := b.CreateSpanBatches(payloads)
	if err != nil {
		return err
	}

	txs := make([]*types.Transaction, 0)

	nonce, err := b.chain.GetNonce(b.batcherAddr)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	for _, frame := range frames {
		var blob eth.Blob
		if err := blob.FromData(frame); err != nil {
			return fmt.Errorf("failed to create blob: %w", err)
		}

		sidecar, blobHashes, err := txmgr.MakeSidecar([]*eth.Blob{&blob})
		if err != nil {
			return fmt.Errorf("failed to create sidecar: %w", err)
		}

		pendingHeader, err := b.chain.GetBlockByHash(parentHash)
		if err != nil {
			return fmt.Errorf("failed to get pending header: %w", err)
		}

		if pendingHeader.ExcessBlobGas() == nil {
			return fmt.Errorf("pending header does not have excess blob gas")
		}

		blobBaseFee := eth.CalcBlobFeeDefault(pendingHeader.Header())
		blobFeeCap := new(uint256.Int).Mul(uint256.NewInt(2), uint256.MustFromBig(blobBaseFee))
		if blobFeeCap.Lt(uint256.NewInt(params.GWei)) { // ensure we meet 1 gwei geth tx-pool minimum
			blobFeeCap = uint256.NewInt(params.GWei)
		}

		txData := &types.BlobTx{
			To:         b.rollupCfg.BatchInboxAddress,
			Data:       nil,
			Gas:        params.TxGas, // intrinsic gas only
			BlobHashes: blobHashes,
			Sidecar:    sidecar,
			ChainID:    uint256.MustFromBig(b.rollupCfg.L1ChainID),
			GasTipCap:  uint256.MustFromBig(big.NewInt(1e9)),
			GasFeeCap:  uint256.MustFromBig(big.NewInt(1e9)),
			BlobFeeCap: blobFeeCap,
			Value:      uint256.NewInt(0),
			Nonce:      nonce,
		}

		nonce++

		signer := types.NewPragueSigner(b.rollupCfg.L1ChainID)

		// sign with batcher key
		tx, err := types.SignNewTx(b.batcherKey, signer, txData)
		if err != nil {
			return fmt.Errorf("failed to sign tx: %w", err)
		}

		txs = append(txs, tx)
	}

	err = b.chain.BuildAndMine(txs)
	if err != nil {
		return fmt.Errorf("failed to build and mine txs: %w", err)
	}

	return nil
}
