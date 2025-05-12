/**
 * Goal: Allow us to intercept certain RPC calls and return a custom response.
 *
 * Example Scenario: We want to intercept eth sendRawTransaction calls, build a
 * block overtime and send it in one call. This would be used to avoid sending the
 * transactions to the mempool for example.
 */

package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/log"
)

type L1ProxyServer struct {
	log    log.Logger
	port   int
	server *http.Server
}

func NewL1ProxyServer(log log.Logger, port int) *L1ProxyServer {
	return &L1ProxyServer{
		log:  log,
		port: port,
	}
}

func (p *L1ProxyServer) Run(ctx context.Context) error {
	// Start the proxy server
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleRequest)

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.port),
		Handler: mux,
	}

	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			p.log.Error("Proxy server error", "err", err)
		}
	}()

	return nil
}

// Stop stops both the proxy server and the underlying client
func (p *L1ProxyServer) Stop() {
	if p.server != nil {
		if err := p.server.Close(); err != nil {
			p.log.Error("Error closing proxy server", "err", err)
		}
	}
}

func (p *L1ProxyServer) ClientURL() string {
	return fmt.Sprintf("http://localhost:%d", p.port)
}

func (p *L1ProxyServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	var request struct {
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      interface{}     `json:"id"`
		JSONRPC string          `json:"jsonrpc"`
	}

	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Error parsing request", http.StatusBadRequest)
		return
	}

	handled, response, err := p.OverrideRequest(request.Method, request.Params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error handling request: %v", err), http.StatusInternalServerError)
		return
	}

	if handled {
		resp := struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      interface{}     `json:"id"`
			Result  json.RawMessage `json:"result"`
		}{
			JSONRPC: request.JSONRPC,
			ID:      request.ID,
			Result:  response,
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			p.log.Error("Error encoding response", "err", err)
		}
		return
	}

	// error 404 if not handled
	http.Error(w, "Method not found", http.StatusNotFound)
}

func (p *L1ProxyServer) OverrideRequest(method string, rawParams json.RawMessage) (bool, json.RawMessage, error) {
	switch method {
	default:
		return true, nil, fmt.Errorf("unsupported method: %s", method)
	}
}
