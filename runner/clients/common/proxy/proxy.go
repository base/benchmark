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
	"sync"

	"github.com/base/base-bench/runner/network/mempool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type ProxyServer struct {
	log        log.Logger
	port       int
	server     *http.Server
	pendingTxs []*ethTypes.Transaction
	clientURL  string
	mempool    *mempool.StaticWorkloadMempool
	nextNonce  map[common.Address]uint64
	mu         sync.Mutex
}

type rpcRequest struct {
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   interface{}     `json:"error,omitempty"`
}

func NewProxyServer(clientURL string, log log.Logger, port int, mempool *mempool.StaticWorkloadMempool) *ProxyServer {
	return &ProxyServer{
		clientURL: clientURL,
		log:       log,
		port:      port,
		mempool:   mempool,
		nextNonce: make(map[common.Address]uint64),
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

func (p *ProxyServer) DrainPendingTxs() []*ethTypes.Transaction {
	p.mu.Lock()
	defer p.mu.Unlock()

	txs := p.pendingTxs
	p.pendingTxs = make([]*ethTypes.Transaction, 0)
	return txs
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

	if len(body) > 0 && body[0] == '[' {
		p.handleBatchRequest(w, body)
		return
	}

	var request rpcRequest

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
		resp := rpcResponse{
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

func (p *ProxyServer) handleBatchRequest(w http.ResponseWriter, body []byte) {
	var requests []rpcRequest
	if err := json.Unmarshal(body, &requests); err != nil {
		http.Error(w, "Error parsing batch request", http.StatusBadRequest)
		return
	}

	responses := make([]rpcResponse, 0, len(requests))
	for _, request := range requests {
		handled, result, err := p.OverrideRequest(request.Method, request.Params)
		response := rpcResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
		}
		if err != nil {
			response.Error = map[string]interface{}{"code": -32000, "message": err.Error()}
		} else if handled {
			response.Result = result
		} else {
			forwardedResponse, err := p.forwardRPCRequest(request)
			if err != nil {
				response.Error = map[string]interface{}{"code": -32000, "message": err.Error()}
			} else {
				response = forwardedResponse
			}
		}
		responses = append(responses, response)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		p.log.Error("Error encoding batch response", "err", err)
	}
}

func (p *ProxyServer) forwardRPCRequest(request rpcRequest) (rpcResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return rpcResponse{}, fmt.Errorf("failed to marshal upstream request: %w", err)
	}

	resp, err := http.Post(p.clientURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return rpcResponse{}, fmt.Errorf("failed to forward request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			p.log.Error("Error closing response body", "err", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return rpcResponse{}, fmt.Errorf("failed to read upstream response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return rpcResponse{}, fmt.Errorf("upstream request returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var forwardedResponse rpcResponse
	if err := json.Unmarshal(respBody, &forwardedResponse); err != nil {
		return rpcResponse{}, fmt.Errorf("failed to decode upstream response: %w", err)
	}
	if forwardedResponse.JSONRPC == "" {
		forwardedResponse.JSONRPC = request.JSONRPC
	}
	if forwardedResponse.ID == nil {
		forwardedResponse.ID = request.ID
	}
	return forwardedResponse, nil
}

func (p *ProxyServer) OverrideRequest(method string, rawParams json.RawMessage) (bool, json.RawMessage, error) {
	switch method {
	case "eth_getTransactionCount":
		var params []string
		if err := json.Unmarshal(rawParams, &params); err != nil {
			return false, nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
		if len(params) == 0 {
			return false, nil, fmt.Errorf("no params found")
		}

		address := common.HexToAddress(params[0])
		nonce, err := p.upstreamTransactionCount(rawParams)
		if err != nil {
			if observedNonce, ok := p.observedTransactionCount(address); ok {
				jsonResponse, _ := json.Marshal(fmt.Sprintf("0x%x", observedNonce))
				return true, jsonResponse, nil
			}
			return false, nil, err
		}
		if observedNonce, ok := p.observedTransactionCount(address); ok && observedNonce > nonce {
			nonce = observedNonce
		}
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

		// Use UnmarshalBinary to support both legacy and typed (EIP-2718) transactions.
		// The previous rlp.DecodeBytes only handled legacy transactions.
		err = tx.UnmarshalBinary(rawTxBytes)

		if err != nil {
			p.log.Error("failed to decode transaction", "err", err)
			return false, nil, fmt.Errorf("failed to decode transaction: %w", err)
		}

		p.recordPendingTransaction(&tx)

		txHash := tx.Hash().Hex()
		jsonResponse, _ := json.Marshal(txHash)
		return true, jsonResponse, nil
	default:
		return false, nil, nil
	}
}

func (p *ProxyServer) upstreamTransactionCount(rawParams json.RawMessage) (uint64, error) {
	body, err := json.Marshal(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_getTransactionCount",
		Params:  rawParams,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal upstream nonce request: %w", err)
	}

	resp, err := http.Post(p.clientURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch upstream transaction count: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			p.log.Error("Error closing response body", "err", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read upstream transaction count response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("upstream transaction count request returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return 0, fmt.Errorf("failed to decode upstream transaction count response: %w", err)
	}
	if rpcResp.Error != nil {
		return 0, fmt.Errorf("upstream transaction count error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	var nonceHex string
	if err := json.Unmarshal(rpcResp.Result, &nonceHex); err != nil {
		return 0, fmt.Errorf("failed to decode upstream transaction count result: %w", err)
	}
	nonce, err := hexutil.DecodeUint64(nonceHex)
	if err != nil {
		return 0, fmt.Errorf("failed to parse upstream transaction count %q: %w", nonceHex, err)
	}
	return nonce, nil
}

func (p *ProxyServer) observedTransactionCount(address common.Address) (uint64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	nonce, ok := p.nextNonce[address]
	return nonce, ok
}

func (p *ProxyServer) recordPendingTransaction(tx *ethTypes.Transaction) {
	from, err := ethTypes.Sender(ethTypes.LatestSignerForChainID(tx.ChainId()), tx)
	if err != nil {
		p.log.Warn("failed to recover sender for observed transaction", "err", err, "hash", tx.Hash())
		p.mu.Lock()
		p.pendingTxs = append(p.pendingTxs, tx)
		p.mu.Unlock()
		return
	}

	nextNonce := tx.Nonce() + 1
	p.mu.Lock()
	p.pendingTxs = append(p.pendingTxs, tx)
	if nextNonce > p.nextNonce[from] {
		p.nextNonce[from] = nextNonce
	}
	p.mu.Unlock()
}

func (p *ProxyServer) DebugResponse(method string, params json.RawMessage, respBody []byte) {
	p.log.Debug("method", "method", method)
	p.log.Debug("params", "params", params)

	if !bytes.HasPrefix(respBody, []byte{0x1f, 0x8b}) {
		p.log.Debug("Response body", "body", string(respBody))
		return
	}

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
