package fakel1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// L1ProxyServer is a proxy server that intercepts RPC and beacon chain requests
// and returns a response from the fake L1 chain.
type L1ProxyServer struct {
	log    log.Logger
	port   int
	server *http.Server

	chain    L1Chain
	chainCfg *params.ChainConfig
}

func NewL1ProxyServer(log log.Logger, port int, chain L1Chain) *L1ProxyServer {
	return &L1ProxyServer{
		log:      log,
		port:     port,
		chain:    chain,
		chainCfg: chain.Genesis().Config,
	}
}

func (p *L1ProxyServer) Run(ctx context.Context) error {
	// Start the proxy server
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleRPCRequest)
	mux.HandleFunc("/eth/v1/beacon/genesis", p.handleBeaconRequestForPath("/eth/v1/beacon/genesis"))
	mux.HandleFunc("/eth/v1/config/spec", p.handleBeaconRequestForPath("/eth/v1/config/spec"))
	mux.HandleFunc("/eth/v1/beacon/blob_sidecars/{slot}", p.handleBeaconRequestForPath("/eth/v1/beacon/blob_sidecars/{slot}"))

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

func (p *L1ProxyServer) handleBeaconRequestForPath(path string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var response json.RawMessage
		switch path {
		case "/eth/v1/beacon/genesis":
			genesis := p.chain.BeaconGenesis()
			response, _ = json.Marshal(genesis)
		case "/eth/v1/config/spec":
			spec := p.chain.ConfigSpec()
			response, _ = json.Marshal(spec)
		case "/eth/v1/beacon/blob_sidecars/{slot}":
			slotStr := r.PathValue("slot")
			log.Debug("Handling blob sidecars request", "slot", slotStr)
			slotInt, err := strconv.Atoi(slotStr)
			if err != nil {
				http.Error(w, "Invalid slot parameter", http.StatusBadRequest)
				return
			}
			slot := uint64(slotInt)
			ctx := context.Background()
			blobSidecars, err := p.chain.GetSidecarsBySlot(ctx, slot)
			if err != nil {
				p.log.Error("Error getting blob sidecars", "slot", slot, "err", err)
				http.Error(w, fmt.Sprintf("Error getting blob sidecars: %v", err), http.StatusInternalServerError)
				return
			}
			response, err = json.Marshal(blobSidecars)
			if err != nil {
				p.log.Error("Error marshaling blob sidecars", "slot", slot, "err", err)
				http.Error(w, fmt.Sprintf("Error marshaling blob sidecars: %v", err), http.StatusInternalServerError)
				return
			}

		default:
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			p.log.Error("Error encoding response", "err", err)
		}
	}
}

func (p *L1ProxyServer) handleRPCRequest(w http.ResponseWriter, r *http.Request) {
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

	response, err := p.OverrideRequest(context.Background(), request.Method, request.Params)
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

func (p *L1ProxyServer) handleGetBlockByHash(ctx context.Context, rawParams json.RawMessage) (json.RawMessage, error) {
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

	if block == nil {
		return nil, fmt.Errorf("block not found for hash: %s", blockHash)
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
}

func (p *L1ProxyServer) handleGetBlockByNumber(_ctx context.Context, rawParams json.RawMessage) (json.RawMessage, error) {
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
}

func (p *L1ProxyServer) handleGetBlockReceipts(ctx context.Context, rawParams json.RawMessage) (json.RawMessage, error) {
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
}

func (p *L1ProxyServer) OverrideRequest(ctx context.Context, method string, rawParams json.RawMessage) (json.RawMessage, error) {
	p.log.Info("got request", "method", method, "params", string([]byte(rawParams)))
	switch method {
	case "eth_getBlockByNumber":
		return p.handleGetBlockByNumber(ctx, rawParams)
	case "eth_getBlockByHash":
		return p.handleGetBlockByHash(ctx, rawParams)
	case "eth_getBlockReceipts":
		return p.handleGetBlockReceipts(ctx, rawParams)
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
}
