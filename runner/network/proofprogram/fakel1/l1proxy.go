/**
 * Goal: Allow us to intercept certain RPC calls and return a custom response.
 *
 * Example Scenario: We want to intercept eth sendRawTransaction calls, build a
 * block overtime and send it in one call. This would be used to avoid sending the
 * transactions to the mempool for example.
 */

package fakel1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type L1ProxyServer struct {
	log    log.Logger
	port   int
	server *http.Server

	chain    *FakeL1Chain
	chainCfg *params.ChainConfig
}

func NewL1ProxyServer(log log.Logger, port int, chain *FakeL1Chain) *L1ProxyServer {
	return &L1ProxyServer{
		log:      log,
		port:     port,
		chain:    chain,
		chainCfg: chain.genesis.Config,
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

	response, err := p.OverrideRequest(context.TODO(), request.Method, request.Params)
	if err != nil {
		p.log.Error("Error handling request", "method", request.Method, "err", err)
		http.Error(w, fmt.Sprintf("Error handling request: %v", err), http.StatusInternalServerError)
		return
	}

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
}

func (p *L1ProxyServer) OverrideRequest(ctx context.Context, method string, rawParams json.RawMessage) (json.RawMessage, error) {
	p.log.Info("got request", "method", method, "params", rawParams)
	switch method {
	case "eth_getBlockByNumber":
		var params []interface{}
		if err := json.Unmarshal(rawParams, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}

		if len(params) != 2 {
			return nil, fmt.Errorf("expected 2 params, got %d", len(params))
		}

		blockNumber, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("expected block number to be a string, got %T", params[0])
		}
		blockNumberInt, err := hexutil.DecodeUint64(blockNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to decode block number: %w", err)
		}

		block, err := p.chain.GetBlockByNumber(blockNumberInt)
		if err != nil {
			return nil, fmt.Errorf("failed to get block by number: %w", err)
		}
		blockJSON, err := json.Marshal(block)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal block: %w", err)
		}

		return blockJSON, nil
	case "eth_getBlockByHash":
		var params []interface{}
		if err := json.Unmarshal(rawParams, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
		if len(params) != 2 {
			return nil, fmt.Errorf("expected 2 params, got %d", len(params))
		}
		blockHash, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("expected block hash to be a string, got %T", params[0])
		}

		includeTxs, ok := params[1].(bool)
		if !ok {
			return nil, fmt.Errorf("expected includeTxs to be a bool, got %T", params[1])
		}

		blockHashBytes, err := hexutil.Decode(blockHash)
		if err != nil {
			return nil, fmt.Errorf("failed to decode block hash: %w", err)
		}
		block, err := p.chain.GetBlockByHash(common.BytesToHash(blockHashBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to get block by hash: %w", err)
		}

		rpcBlock, err := RPCMarshalBlock(ctx, block, true, includeTxs, p.chainCfg, p.chain)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal block: %w", err)
		}

		blockJSON, err := json.Marshal(rpcBlock)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal block: %w", err)
		}
		return blockJSON, nil
	case "eth_getBlockReceipts":
		var params []interface{}
		if err := json.Unmarshal(rawParams, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}

		if len(params) != 1 {
			return nil, fmt.Errorf("expected 1 param, got %d", len(params))
		}

		blockHash, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("expected block hash to be a string, got %T", params[0])
		}

		blockHashBytes, err := hexutil.Decode(blockHash)
		if err != nil {
			return nil, fmt.Errorf("failed to decode block hash: %w", err)
		}

		receipts, err := p.chain.GetReceipts(ctx, common.BytesToHash(blockHashBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to get receipts: %w", err)
		}

		return json.Marshal(receipts)
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
}
