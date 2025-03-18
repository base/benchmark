package common

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	RPCTimeout    = 60 * time.Second
	RetryInterval = time.Second
)

// WaitForRPC waits for the ETH EL RPC server to be ready.
func WaitForRPC(ctx context.Context, client *ethclient.Client) error {
	ready := false
	var lastErr error

	for i := uint(0); i < uint(RPCTimeout/RetryInterval); i++ {
		_, err := client.BlockNumber(ctx)
		if err == nil {
			ready = true
			break
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		time.Sleep(RetryInterval)
	}

	if !ready {
		return lastErr
	}

	return nil
}
