package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/base/base-bench/runner/clients/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
)

// flashblocksClient implements the FlashblocksClient interface for collecting
// flashblock payloads from the builder via websocket and broadcasting them to listeners.
type flashblocksClient struct {
	log       log.Logger
	port      uint64
	conn      *websocket.Conn
	listeners []types.FlashblockListener
	mu        sync.RWMutex
	stopChan  chan struct{}
	stopOnce  sync.Once
}

// NewFlashblocksClient creates a new flashblocks websocket client.
func NewFlashblocksClient(log log.Logger, port uint64) types.FlashblocksClient {
	return &flashblocksClient{
		log:       log,
		port:      port,
		listeners: make([]types.FlashblockListener, 0),
		stopChan:  make(chan struct{}),
	}
}

// Start begins collecting flashblocks from the websocket connection.
// This method connects to the websocket in a goroutine to avoid blocking.
func (f *flashblocksClient) Start(ctx context.Context) error {
	url := fmt.Sprintf("ws://localhost:%d", f.port)
	f.log.Info("Connecting to flashblocks websocket", "url", url)

	// Use a separate context with timeout for the connection attempt
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(connectCtx, url, nil)
	if err != nil {
		return err
	}

	f.mu.Lock()
	f.conn = conn
	f.mu.Unlock()

	f.log.Info("Connected to flashblocks websocket", "url", url)

	// Read messages in this goroutine
	go f.readMessages(ctx)

	return nil
}

// readMessages reads flashblock messages from the websocket in a blocking loop.
// It exits on any error or when the context is cancelled.
func (f *flashblocksClient) readMessages(ctx context.Context) {
	defer func() {
		f.mu.Lock()
		if f.conn != nil {
			err := f.conn.Close()
			if err != nil {
				f.log.Warn("Error closing flashblocks websocket connection", "err", err)
			}
			f.conn = nil
		}
		f.mu.Unlock()
	}()

	// Channel to signal when a message is received or error occurs
	type readResult struct {
		message []byte
		err     error
	}
	readChan := make(chan readResult, 1)

	// Start a goroutine to read from the websocket
	go func() {
		for {
			_, message, err := f.conn.ReadMessage()
			readChan <- readResult{message: message, err: err}
			if err != nil {
				// Exit on any error to avoid panic on repeated reads
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			f.log.Debug("Context cancelled, stopping flashblocks client")
			return
		case <-f.stopChan:
			f.log.Debug("Stop signal received, stopping flashblocks client")
			return
		case result := <-readChan:
			if result.err != nil {
				// Check if this is a normal closure
				if websocket.IsCloseError(result.err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					f.log.Debug("Flashblocks websocket closed normally")
				}
				return
			}

			// Deserialize flashblock payload
			var flashblock types.FlashblocksPayloadV1
			if err := json.Unmarshal(result.message, &flashblock); err != nil {
				f.log.Warn("Failed to deserialize flashblock payload", "err", err)
				continue
			}

			// Log flashblock details at DEBUG level
			txCount := len(flashblock.Diff.Transactions)
			f.log.Debug("Received flashblock",
				"payload_id", fmt.Sprintf("%x", flashblock.PayloadID),
				"index", flashblock.Index,
				"tx_count", txCount,
				"gas_used", flashblock.Diff.GasUsed,
				"block_hash", flashblock.Diff.BlockHash.Hex(),
			)

			// Broadcast to all listeners
			f.broadcastFlashblock(flashblock)
		}
	}
}

// Stop stops collection and closes the websocket connection.
// This method is idempotent and can be called multiple times safely.
func (f *flashblocksClient) Stop() error {
	// Use sync.Once to ensure cleanup only happens once
	var stopErr error
	f.stopOnce.Do(func() {
		f.log.Info("Stopping flashblocks client")

		// Signal the read goroutine to stop
		close(f.stopChan)

		f.mu.Lock()
		defer f.mu.Unlock()

		if f.conn != nil {
			// Send close message
			err := f.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				f.log.Warn("Error sending close message to flashblocks websocket", "err", err)
			}

			// Close the connection
			err = f.conn.Close()
			if err != nil {
				f.log.Warn("Error closing flashblocks websocket connection", "err", err)
				stopErr = err
			}
			f.conn = nil
		}

		f.log.Info("Flashblocks client stopped")
	})

	return stopErr
}

// AddListener registers a listener to receive flashblock payloads.
func (f *flashblocksClient) AddListener(listener types.FlashblockListener) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listeners = append(f.listeners, listener)
	f.log.Debug("Added flashblock listener", "total_listeners", len(f.listeners))
}

// RemoveListener unregisters a listener.
func (f *flashblocksClient) RemoveListener(listener types.FlashblockListener) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i, l := range f.listeners {
		if l == listener {
			f.listeners = append(f.listeners[:i], f.listeners[i+1:]...)
			f.log.Debug("Removed flashblock listener", "total_listeners", len(f.listeners))
			return
		}
	}
}

// broadcastFlashblock sends the flashblock to all registered listeners.
func (f *flashblocksClient) broadcastFlashblock(flashblock types.FlashblocksPayloadV1) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, listener := range f.listeners {
		// Call each listener (synchronously for now)
		listener.OnFlashblock(flashblock)
	}
}

// IsConnected returns true if the websocket connection is active.
func (f *flashblocksClient) IsConnected() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.conn != nil
}
