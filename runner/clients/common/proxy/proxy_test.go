package proxy

import (
	"bytes"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/base/base-bench/runner/network/mempool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

func TestHandleBatchRequestCapturesRawTransactions(t *testing.T) {
	chainID := big.NewInt(8453)
	tx := signedTestTx(t, chainID)
	rawTx, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal tx: %v", err)
	}

	server := NewProxyServer(
		"http://127.0.0.1:8545",
		log.New(),
		0,
		mempool.NewStaticWorkloadMempool(log.New(), chainID),
	)

	body, err := json.Marshal([]map[string]any{
		{
			"jsonrpc": "2.0",
			"id":      0,
			"method":  "eth_sendRawTransaction",
			"params":  []string{hexutil.Encode(rawTx)},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.handleRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var responses []struct {
		Result string         `json:"result"`
		Error  map[string]any `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &responses); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	if responses[0].Error != nil {
		t.Fatalf("expected successful response, got error %v", responses[0].Error)
	}
	if responses[0].Result != tx.Hash().Hex() {
		t.Fatalf("expected tx hash %s, got %s", tx.Hash().Hex(), responses[0].Result)
	}

	pending := server.DrainPendingTxs()
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending tx, got %d", len(pending))
	}
	if pending[0].Hash() != tx.Hash() {
		t.Fatalf("expected pending tx %s, got %s", tx.Hash(), pending[0].Hash())
	}
}

func TestHandleBatchRequestForwardsPassThroughMethods(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode upstream request: %v", err)
		}
		if req.Method != "eth_chainId" {
			t.Fatalf("expected eth_chainId, got %s", req.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "0x2105",
		}); err != nil {
			t.Fatalf("encode upstream response: %v", err)
		}
	}))
	defer upstream.Close()

	server := NewProxyServer(
		upstream.URL,
		log.New(),
		0,
		mempool.NewStaticWorkloadMempool(log.New(), big.NewInt(8453)),
	)

	body, err := json.Marshal([]map[string]any{
		{
			"jsonrpc": "2.0",
			"id":      7,
			"method":  "eth_chainId",
			"params":  []string{},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.handleRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var responses []struct {
		Result string         `json:"result"`
		Error  map[string]any `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &responses); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	if responses[0].Error != nil {
		t.Fatalf("expected successful response, got error %v", responses[0].Error)
	}
	if responses[0].Result != "0x2105" {
		t.Fatalf("expected forwarded result 0x2105, got %s", responses[0].Result)
	}
}

func TestHandleBatchRequestSupportsMixedForwardAndCapture(t *testing.T) {
	chainID := big.NewInt(8453)
	tx := signedTestTx(t, chainID)
	rawTx, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal tx: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode upstream request: %v", err)
		}
		if req.Method != "eth_gasPrice" {
			t.Fatalf("expected eth_gasPrice, got %s", req.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "0x3b9aca00",
		}); err != nil {
			t.Fatalf("encode upstream response: %v", err)
		}
	}))
	defer upstream.Close()

	server := NewProxyServer(
		upstream.URL,
		log.New(),
		0,
		mempool.NewStaticWorkloadMempool(log.New(), chainID),
	)

	body, err := json.Marshal([]map[string]any{
		{
			"jsonrpc": "2.0",
			"id":      0,
			"method":  "eth_gasPrice",
			"params":  []string{},
		},
		{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "eth_sendRawTransaction",
			"params":  []string{hexutil.Encode(rawTx)},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.handleRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var responses []struct {
		Result string         `json:"result"`
		Error  map[string]any `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &responses); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}
	if responses[0].Error != nil {
		t.Fatalf("expected successful forwarded response, got error %v", responses[0].Error)
	}
	if responses[0].Result != "0x3b9aca00" {
		t.Fatalf("expected forwarded gas price, got %s", responses[0].Result)
	}
	if responses[1].Error != nil {
		t.Fatalf("expected successful captured tx response, got error %v", responses[1].Error)
	}
	if responses[1].Result != tx.Hash().Hex() {
		t.Fatalf("expected tx hash %s, got %s", tx.Hash().Hex(), responses[1].Result)
	}

	pending := server.DrainPendingTxs()
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending tx, got %d", len(pending))
	}
	if pending[0].Hash() != tx.Hash() {
		t.Fatalf("expected pending tx %s, got %s", tx.Hash(), pending[0].Hash())
	}
}

func TestGetTransactionCountForwardsUpstreamNonce(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0xfa",
		}); err != nil {
			t.Fatalf("encode upstream response: %v", err)
		}
	}))
	defer upstream.Close()

	server := NewProxyServer(
		upstream.URL,
		log.New(),
		0,
		mempool.NewStaticWorkloadMempool(log.New(), big.NewInt(8453)),
	)

	result := callProxyRPC(t, server, "eth_getTransactionCount", []string{common.Address{1}.Hex(), "pending"})
	if result != "0xfa" {
		t.Fatalf("expected upstream nonce 0xfa, got %s", result)
	}
}

func TestGetTransactionCountIncludesObservedPendingTransactions(t *testing.T) {
	chainID := big.NewInt(8453)
	tx := signedTestTxWithNonce(t, chainID, 250)
	from, err := types.Sender(types.LatestSignerForChainID(chainID), tx)
	if err != nil {
		t.Fatalf("recover sender: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0xfa",
		}); err != nil {
			t.Fatalf("encode upstream response: %v", err)
		}
	}))
	defer upstream.Close()

	server := NewProxyServer(
		upstream.URL,
		log.New(),
		0,
		mempool.NewStaticWorkloadMempool(log.New(), chainID),
	)

	rawTx, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal tx: %v", err)
	}
	callProxyRPC(t, server, "eth_sendRawTransaction", []string{hexutil.Encode(rawTx)})

	result := callProxyRPC(t, server, "eth_getTransactionCount", []string{from.Hex(), "pending"})
	if result != "0xfb" {
		t.Fatalf("expected observed nonce 0xfb, got %s", result)
	}
}

func callProxyRPC(t *testing.T, server *ProxyServer, method string, params any) string {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.handleRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Result string         `json:"result"`
		Error  map[string]any `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("expected successful response, got error %v", response.Error)
	}
	return response.Result
}

func signedTestTx(t *testing.T, chainID *big.Int) *types.Transaction {
	return signedTestTxWithNonce(t, chainID, 0)
}

func signedTestTxWithNonce(t *testing.T, chainID *big.Int, nonce uint64) *types.Transaction {
	t.Helper()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Gas:       21_000,
		To:        &common.Address{1},
		Value:     big.NewInt(1),
	})

	signed, err := types.SignTx(tx, types.NewIsthmusSigner(chainID), key)
	if err != nil {
		t.Fatalf("sign tx: %v", err)
	}
	return signed
}
