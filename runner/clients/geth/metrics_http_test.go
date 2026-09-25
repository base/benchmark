package geth

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/base/base-bench/runner/metrics"
	"github.com/ethereum/go-ethereum/log"
)

func TestMetricsCollectorCollect(t *testing.T) {
	const body = "{\"chain/execution.50-percentile\": 42}"
	for _, stage := range []string{"success", "waiting_for_headers", "reading_body"} {
		t.Run(stage, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if stage == "reading_body" {
					_, _ = io.WriteString(w, body[:len(body)/2])
					w.(http.Flusher).Flush()
				}
				close(started)
				if stage != "success" {
					select {
					case <-r.Context().Done():
					case <-release:
					}
					return
				}
				_, _ = io.WriteString(w, body)
			}))
			t.Cleanup(func() {
				// Release the handler even if a broken collector ignores cancellation.
				close(release)
				server.Close()
			})

			port := server.Listener.Addr().(*net.TCPAddr).Port
			collector := newMetricsCollector(log.New(), nil, port)
			block := metrics.NewBlockMetrics()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- collector.Collect(ctx, block) }()

			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("metrics request did not reach the server")
			}
			if stage != "success" {
				cancel()
			}
			var err error
			select {
			case err = <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("Collect did not return after cancellation")
			}
			if stage != "success" {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("Collect error = %v, want context.Canceled", err)
				}
				if len(collector.GetMetrics()) != 0 {
					t.Fatal("cancelled scrape recorded a block")
				}
				return
			}
			if err != nil {
				t.Fatalf("Collect: %v", err)
			}
			if got := block.ExecutionMetrics["chain/execution.50-percentile"]; got != float64(42) {
				t.Fatalf("collected metric = %v, want 42", got)
			}
			if got := collector.GetMetrics(); len(got) != 1 || got[0].ExecutionMetrics["chain/execution.50-percentile"] != float64(42) {
				t.Fatalf("recorded metrics = %v, want one block containing 42", got)
			}
		})
	}
}
