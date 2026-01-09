package types

import "context"

// FlashblockListener receives flashblock payloads as they are received from the websocket.
type FlashblockListener interface {
	// OnFlashblock is called when a new flashblock payload is received
	OnFlashblock(flashblock FlashblocksPayloadV1)
}

// FlashblocksClient is an interface for collecting flashblock payloads from a websocket connection.
// Only clients that support flashblocks (e.g., rbuilder) will provide a non-nil implementation.
// It uses a broadcast pattern where listeners can subscribe to receive flashblock updates.
type FlashblocksClient interface {
	// Start begins collecting flashblocks from the websocket connection
	Start(ctx context.Context) error

	// Stop stops collection and closes the websocket connection
	Stop() error

	// AddListener registers a listener to receive flashblock payloads
	AddListener(listener FlashblockListener)

	// RemoveListener unregisters a listener
	RemoveListener(listener FlashblockListener)

	// IsConnected returns true if the websocket connection is active
	IsConnected() bool
}
