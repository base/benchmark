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

func signedTestTx(t *testing.T, chainID *big.Int) *types.Transaction {
	t.Helper()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     0,
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
