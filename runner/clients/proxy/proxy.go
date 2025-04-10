/**
 * Goal: Allow us to intercept certain RPC calls and return a custom response.
 *
 * Example Scenario: We want to intercept eth sendRawTransaction calls, build a
 * block overtime and send it in one call. This would be used to avoid sending the
 * transactions to the mempool for example.
 */

package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/base/base-bench/runner/clients/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type ProxyServer struct {
	client types.ExecutionClient
	log    log.Logger
	port   int
	server *http.Server
}

func NewProxyServer(client types.ExecutionClient, log log.Logger, port int) *ProxyServer {
	return &ProxyServer{
		client: client,
		log:    log,
		port:   port,
	}
}

func (p *ProxyServer) Run(ctx context.Context, config *types.RuntimeConfig) error {
	if err := p.client.Run(ctx, config); err != nil {
		return err
	}

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
func (p *ProxyServer) Stop() {
	if p.server != nil {
		if err := p.server.Close(); err != nil {
			p.log.Error("Error closing proxy server", "err", err)
		}
	}
	p.client.Stop()
}

func (p *ProxyServer) Client() *ethclient.Client {
	return p.client.Client()
}

func (p *ProxyServer) ClientURL() string {
	return fmt.Sprintf("http://localhost:%d", p.port)
}

func (p *ProxyServer) AuthClient() client.RPC {
	return p.client.AuthClient()
}

func (p *ProxyServer) handleRequest(w http.ResponseWriter, r *http.Request) {
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

	client := &http.Client{}
	req, err := http.NewRequest("POST", p.client.ClientURL(), bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}
	req.Header = r.Header

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error forwarding request", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			p.log.Error("Error closing response body", "err", err)
		}
	}()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		p.log.Error("Error copying response body", "err", err)
	}
}

func (p *ProxyServer) OverrideRequest(method string, params json.RawMessage) (bool, json.RawMessage, error) {
	switch method {
	// Example of how to intercept a request
	// case "eth_getBlockByNumber":
	// 	response := "0x100"
	// 	return true, json.RawMessage(`"` + response + `"`), nil

	default:
		return false, nil, nil
	}
}
