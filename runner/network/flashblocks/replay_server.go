package flashblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/base/base-bench/runner/clients/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
)

// ReplayServer replays flashblock payloads to connected clients via websocket.
type ReplayServer struct {
	log         log.Logger
	port        uint64
	flashblocks map[uint64][]types.FlashblocksPayloadV1
	blockTime   time.Duration

	server   *http.Server
	upgrader websocket.Upgrader

	mu          sync.RWMutex
	connections []*websocket.Conn
	started     bool
	stopChan    chan struct{}
	stopOnce    sync.Once
}

func NewReplayServer(log log.Logger, port uint64, flashblocks map[uint64][]types.FlashblocksPayloadV1, blockTime time.Duration) *ReplayServer {
	return &ReplayServer{
		log:         log,
		port:        port,
		flashblocks: flashblocks,
		blockTime:   blockTime,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		connections: make([]*websocket.Conn, 0),
		stopChan:    make(chan struct{}),
	}
}

func (s *ReplayServer) URL() string {
	return fmt.Sprintf("ws://127.0.0.1:%d", s.port)
}

func (s *ReplayServer) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("server already started")
	}
	s.started = true
	s.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWebSocket)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	go func() {
		s.log.Info("Starting flashblock replay server", "port", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("Flashblock replay server error", "err", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	return nil
}

func (s *ReplayServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("Failed to upgrade websocket connection", "err", err)
		return
	}

	s.mu.Lock()
	s.connections = append(s.connections, conn)
	s.mu.Unlock()

	s.log.Info("New client connected to flashblock replay server")

	for {
		select {
		case <-s.stopChan:
			return
		default:
			if _, _, err := conn.ReadMessage(); err != nil {
				s.removeConnection(conn)
				return
			}
		}
	}
}

func (s *ReplayServer) removeConnection(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.connections {
		if c == conn {
			s.connections = append(s.connections[:i], s.connections[i+1:]...)
			_ = conn.Close()
			s.log.Debug("Client disconnected from flashblock replay server")
			return
		}
	}
}

// ReplayFlashblocks replays flashblocks to connected clients at evenly spaced intervals.
func (s *ReplayServer) ReplayFlashblock(ctx context.Context, blockNumber uint64) error {
	if len(s.flashblocks) == 0 {
		s.log.Info("No flashblocks to replay")
		return nil
	}

	flashblocks, ok := s.flashblocks[blockNumber]
	if !ok {
		s.log.Info("No flashblocks to replay for block", "block_number", blockNumber)
		return nil
	}

	s.log.Info("Starting flashblock replay",
		"flashblocks", len(flashblocks),
	)

	numIntervals := 1
	if len(flashblocks) > 1 {
		numIntervals = len(flashblocks)
	}

	interval := s.blockTime / time.Duration(numIntervals)

	s.log.Debug("Replaying flashblocks for block",
		"block_number", blockNumber,
		"num_flashblocks", len(flashblocks),
		"interval", interval,
	)

	for i, flashblock := range flashblocks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := s.broadcastFlashblock(flashblock); err != nil {
			s.log.Warn("Error broadcasting flashblock", "err", err, "index", i)
		}

		time.Sleep(interval)
	}

	s.log.Info("Flashblock replay complete")
	return nil
}


func (s *ReplayServer) broadcastFlashblock(flashblock types.FlashblocksPayloadV1) error {
	data, err := json.Marshal(flashblock)
	if err != nil {
		return fmt.Errorf("failed to marshal flashblock: %w", err)
	}

	s.mu.RLock()
	connections := make([]*websocket.Conn, len(s.connections))
	copy(connections, s.connections)
	s.mu.RUnlock()

	var lastErr error
	for _, conn := range connections {
		// Use BinaryMessage - base-reth-node requires binary websocket messages
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			s.log.Warn("Failed to send flashblock to client", "err", err)
			lastErr = err
		}
	}

	blockNumber := 0
	if flashblock.Base != nil {
		blockNumber = int(flashblock.Base.BlockNumber)
	}

	s.log.Debug("Broadcasted flashblock",
		"payload_id", fmt.Sprintf("%x", flashblock.PayloadID),
		"index", flashblock.Index,
		"block_number", blockNumber,
		"num_clients", len(connections),
	)

	return lastErr
}

// Stop stops the server. Safe to call multiple times.
func (s *ReplayServer) Stop() error {
	var stopErr error

	s.stopOnce.Do(func() {
		s.mu.Lock()
		if !s.started {
			s.mu.Unlock()
			return
		}
		s.started = false
		s.mu.Unlock()

		s.log.Info("Stopping flashblock replay server")

		close(s.stopChan)

		s.mu.Lock()
		for _, conn := range s.connections {
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			_ = conn.Close()
		}
		s.connections = nil
		s.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if s.server != nil {
			if err := s.server.Shutdown(ctx); err != nil {
				s.log.Warn("Error shutting down flashblock replay server", "err", err)
				stopErr = err
				return
			}
		}

		s.log.Info("Flashblock replay server stopped")
	})

	return stopErr
}

func (s *ReplayServer) WaitForConnection(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		s.mu.RLock()
		numConnections := len(s.connections)
		s.mu.RUnlock()

		if numConnections > 0 {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for client connection")
}
