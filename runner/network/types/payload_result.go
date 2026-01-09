package types

import (
	clientTypes "github.com/base/base-bench/runner/clients/types"
	"github.com/ethereum/go-ethereum/beacon/engine"
)

// PayloadResult contains the results from a sequencer benchmark run, including
// both the executable payloads and any flashblock payloads that were collected.
type PayloadResult struct {
	// ExecutablePayloads are the execution payloads generated during the benchmark
	ExecutablePayloads []engine.ExecutableData

	// Flashblocks are the flashblock payloads collected during the benchmark (if available)
	Flashblocks []clientTypes.FlashblocksPayloadV1
}

// HasFlashblocks returns true if flashblock payloads were collected.
func (p *PayloadResult) HasFlashblocks() bool {
	return len(p.Flashblocks) > 0
}
