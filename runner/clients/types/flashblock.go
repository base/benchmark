package types

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// ExecutionPayloadFlashblockDeltaV1 represents the modified portions of an execution payload
// within a flashblock. This structure contains only the fields that can be updated during
// block construction, such as state root, receipts, logs, and new transactions.
type ExecutionPayloadFlashblockDeltaV1 struct {
	// StateRoot is the state root of the block
	StateRoot common.Hash `json:"state_root"`
	// ReceiptsRoot is the receipts root of the block
	ReceiptsRoot common.Hash `json:"receipts_root"`
	// LogsBloom is the logs bloom of the block
	LogsBloom types.Bloom `json:"logs_bloom"`
	// GasUsed is the gas used of the block
	GasUsed hexutil.Uint64 `json:"gas_used"`
	// BlockHash is the block hash of the block
	BlockHash common.Hash `json:"block_hash"`
	// Transactions are the transactions of the block
	Transactions []hexutil.Bytes `json:"transactions"`
	// Withdrawals are the withdrawals enabled with V2
	Withdrawals []Withdrawal `json:"withdrawals"`
	// WithdrawalsRoot is the withdrawals root of the block
	WithdrawalsRoot common.Hash `json:"withdrawals_root"`
	// BlobGasUsed is the blob gas used
	BlobGasUsed *hexutil.Uint64 `json:"blob_gas_used,omitempty"`
}

// Withdrawal represents a validator withdrawal
type Withdrawal struct {
	Index     hexutil.Uint64 `json:"index"`
	Validator hexutil.Uint64 `json:"validator"`
	Address   common.Address `json:"address"`
	Amount    hexutil.Uint64 `json:"amount"`
}

// ExecutionPayloadBaseV1 represents the base configuration of an execution payload that
// remains constant throughout block construction. This includes fundamental block properties
// like parent hash, block number, and other header fields that are determined at block
// creation and cannot be modified.
type ExecutionPayloadBaseV1 struct {
	// ParentBeaconBlockRoot is the Ecotone parent beacon block root
	ParentBeaconBlockRoot common.Hash `json:"parent_beacon_block_root"`
	// ParentHash is the parent hash of the block
	ParentHash common.Hash `json:"parent_hash"`
	// FeeRecipient is the fee recipient of the block
	FeeRecipient common.Address `json:"fee_recipient"`
	// PrevRandao is the previous randao of the block
	PrevRandao common.Hash `json:"prev_randao"`
	// BlockNumber is the block number
	BlockNumber hexutil.Uint64 `json:"block_number"`
	// GasLimit is the gas limit of the block
	GasLimit hexutil.Uint64 `json:"gas_limit"`
	// Timestamp is the timestamp of the block
	Timestamp hexutil.Uint64 `json:"timestamp"`
	// ExtraData is the extra data of the block
	ExtraData hexutil.Bytes `json:"extra_data"`
	// BaseFeePerGas is the base fee per gas of the block
	BaseFeePerGas *hexutil.Big `json:"base_fee_per_gas"`
}

// PayloadID is a unique identifier for a payload
type PayloadID [8]byte

// MarshalJSON implements json.Marshaler for PayloadID
func (p PayloadID) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Bytes(p[:]))
}

// UnmarshalJSON implements json.Unmarshaler for PayloadID
func (p *PayloadID) UnmarshalJSON(data []byte) error {
	var b hexutil.Bytes
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	if len(b) != 8 {
		return json.Unmarshal(data, &b)
	}
	copy(p[:], b)
	return nil
}

// FlashblocksPayloadV1 represents a flashblock payload containing the base execution
// payload configuration and the delta/diff containing modified portions.
type FlashblocksPayloadV1 struct {
	// PayloadID is the payload id of the flashblock
	PayloadID PayloadID `json:"payload_id"`
	// Index is the index of the flashblock in the block
	Index uint64 `json:"index"`
	// Base is the base execution payload configuration (optional, only present in first flashblock)
	Base *ExecutionPayloadBaseV1 `json:"base,omitempty"`
	// Diff is the delta/diff containing modified portions of the execution payload
	Diff ExecutionPayloadFlashblockDeltaV1 `json:"diff"`
	// Metadata is additional metadata associated with the flashblock
	Metadata json.RawMessage `json:"metadata"`
}
