package payload

import (
	"context"
	"errors"
	"strings"

	"github.com/base/base-bench/runner/benchmark"
	clienttypes "github.com/base/base-bench/runner/clients/types"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/contract"
	"github.com/base/base-bench/runner/payload/transferonly"
	"github.com/base/base-bench/runner/payload/txfuzz"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum/go-ethereum/log"
)

func NewPayloadWorker(ctx context.Context, log log.Logger, testConfig *benchtypes.TestConfig, sequencerClient clienttypes.ExecutionClient, payloadType benchmark.TransactionPayload) (worker.Worker, error) {
	config := testConfig.Config
	genesis := testConfig.Genesis

	params := testConfig.Params

	privateKey := testConfig.PrefundPrivateKey
	amount := &testConfig.PrefundAmount

	var worker worker.Worker
	var err error

	switch {
	case payloadType == "tx-fuzz":
		worker, err = txfuzz.NewTxFuzzPayloadWorker(
			log, sequencerClient.ClientURL(), params, privateKey, amount, config.TxFuzzBinary())
	case payloadType == "transfer-only":
		worker, err = transferonly.NewTransferPayloadWorker(
			ctx, log, sequencerClient.ClientURL(), params, privateKey, amount, &genesis)
	case strings.HasPrefix(string(payloadType), "contract"):
		worker, err = contract.NewContractPayloadWorker(
			log, sequencerClient.ClientURL(), params, privateKey, amount, &genesis, payloadType, config)
	default:
		return nil, errors.New("invalid payload type")
	}

	return worker, err
}
