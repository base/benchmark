package payload

import (
	"context"
	"errors"

	clienttypes "github.com/base/base-bench/runner/clients/types"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/contract"
	"github.com/base/base-bench/runner/payload/transferonly"
	"github.com/base/base-bench/runner/payload/txfuzz"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/yaml.v3"
)

func NewPayloadWorker(ctx context.Context, log log.Logger, testConfig *benchtypes.TestConfig, sequencerClient clienttypes.ExecutionClient, definition Definition) (worker.Worker, error) {
	config := testConfig.Config
	genesis := testConfig.Genesis

	params := testConfig.Params

	privateKey := testConfig.PrefundPrivateKey
	amount := &testConfig.PrefundAmount

	var worker worker.Worker
	var err error

	switch definition.Type {
	case "tx-fuzz":
		worker, err = txfuzz.NewTxFuzzPayloadWorker(
			log, sequencerClient.ClientURL(), params, privateKey, amount, config.TxFuzzBinary(), genesis.Config.ChainID)
	case "transfer-only":
		worker, err = transferonly.NewTransferPayloadWorker(
			ctx, log, sequencerClient.ClientURL(), params, privateKey, amount, &genesis, definition.Params)
	case "contract":
		worker, err = contract.NewContractPayloadWorker(
			log, sequencerClient.ClientURL(), params, privateKey, amount, &genesis, config, definition.Params)
	default:
		return nil, errors.New("invalid payload type")
	}

	return worker, err
}

type Definition struct {
	Name   *string `yaml:"name"`
	ID     string  `yaml:"id"`
	Type   string  `yaml:"type"`
	Params any     `yaml:"-"`
}

func (t *Definition) UnmarshalYAML(node *yaml.Node) error {
	type txPayloadWithoutParams struct {
		Name string `yaml:"name"`
		ID   string `yaml:"id"`
		Type string `yaml:"type"`
	}

	var txPayload txPayloadWithoutParams
	err := node.Decode(&txPayload)
	if err != nil {
		return err
	}

	t.Name = &txPayload.Name
	t.ID = txPayload.ID
	t.Type = txPayload.Type

	params := interface{}(nil)
	switch t.Type {
	case "transfer-only":
		params = &transferonly.TransferOnlyPayloadDefinition{}
	case "tx-fuzz":
		params = &txfuzz.TxFuzzPayloadDefinition{}
	case "contract":
		params = &contract.ContractPayloadDefinition{}
	}

	err = node.Decode(params)
	if err != nil {
		return err
	}

	t.Params = params
	return nil
}
