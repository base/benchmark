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
	"compress/gzip"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/base/base-bench/runner/network/mempool"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

type ProxyServer struct {
	log        log.Logger
	port       int
	server     *http.Server
	pendingTxs []*ethTypes.Transaction
	clientURL  string
	mempool    *mempool.StaticWorkloadMempool
}

func NewProxyServer(clientURL string, log log.Logger, port int, mempool *mempool.StaticWorkloadMempool) *ProxyServer {
	return &ProxyServer{
		clientURL: clientURL,
		log:       log,
		port:      port,
		mempool:   mempool,
	}
}

func (p *ProxyServer) Run(ctx context.Context) error {
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

func (p *ProxyServer) PendingTxs() []*ethTypes.Transaction {
	return p.pendingTxs
}

func (p *ProxyServer) ClearPendingTxs() {
	p.pendingTxs = make([]*ethTypes.Transaction, 0)
}

// Stop stops both the proxy server and the underlying client
func (p *ProxyServer) Stop() {
	if p.server != nil {
		if err := p.server.Close(); err != nil {
			p.log.Error("Error closing proxy server", "err", err)
		}
	}
}

func (p *ProxyServer) ClientURL() string {
	return fmt.Sprintf("http://localhost:%d", p.port)
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
	req, err := http.NewRequest("POST", p.clientURL, bytes.NewReader(body))
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		p.log.Error("Error reading response body", "err", err)
		return
	}

	_, err = w.Write(respBody)
	if err != nil {
		p.log.Error("Error copying response body", "err", err)
	}

	p.DebugResponse(request.Method, request.Params, respBody)
}

func (p *ProxyServer) OverrideRequest(method string, rawParams json.RawMessage) (bool, json.RawMessage, error) {
	switch method {
	case "eth_getTransactionCount":
		var params []string
		if err := json.Unmarshal(rawParams, &params); err != nil {
			return false, nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}

		nonce := p.mempool.GetTransactionCount(common.HexToAddress(params[0]))
		jsonResponse, _ := json.Marshal(fmt.Sprintf("0x%x", nonce))
		return true, jsonResponse, nil

	case "eth_sendRawTransaction":
		var params []string
		if err := json.Unmarshal(rawParams, &params); err != nil {
			return false, nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}

		if len(params) == 0 {
			return false, nil, fmt.Errorf("no params found")
		}

		var tx ethTypes.Transaction

		rawTxHex := params[0]
		rawTxBytes, err := hex.DecodeString(rawTxHex[2:]) // strip "0x"
		if err != nil {
			p.log.Error("failed to decode hex", "err", err)
			return false, nil, fmt.Errorf("failed to decode hex: %w", err)
		}

		err = rlp.DecodeBytes(rawTxBytes, &tx)

		if err != nil {
			p.log.Error("failed to decode RLP", "err", err)
			return false, nil, fmt.Errorf("failed to decode RLP: %w", err)
		}

		p.pendingTxs = append(p.pendingTxs, &tx)

		txHash := tx.Hash().Hex()
		jsonResponse, _ := json.Marshal(txHash)
		return true, jsonResponse, nil
	default:
		return false, nil, nil
	}
}

func (p *ProxyServer) DebugResponse(method string, params json.RawMessage, respBody []byte) {
	p.log.Debug("method", "method", method)
	p.log.Debug("params", "params", params)

	gzipReader, err := gzip.NewReader(bytes.NewReader(respBody))
	if err != nil {
		p.log.Error("Error creating gzip reader", "err", err)
		return
	}
	defer func() {
		if err := gzipReader.Close(); err != nil {
			p.log.Error("Error closing gzip reader", "err", err)
		}
	}()

	uncompressedBody, err := io.ReadAll(gzipReader)

	if err != nil {
		p.log.Error("Error reading uncompressed response body", "err", err)
		return
	}
	p.log.Debug("Uncompressed body", "body", string(uncompressedBody))
}
