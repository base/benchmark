package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/base/base-bench/runner/payload/simulator/simulatorstats"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:     "rpc-url",
		Usage:    "RPC URL of the chain to fetch payloads from",
		Required: true,
	},
	&cli.IntFlag{
		Name:  "sample-size",
		Usage: "Number of payloads to sample",
		Value: 10,
	},
	&cli.StringFlag{
		Name:  "genesis",
		Usage: "Genesis JSON file",
		Value: "genesis.json",
	},
	&cli.StringFlag{
		Name:  "chain-id",
		Usage: "Chain ID to load genesis from",
		Value: "",
	},
}

func init() {
	flags = append(flags, log.CLIFlags("SIM")...)
}

func main() {
	app := cli.NewApp()
	app.Name = "payload-simulator"
	app.Usage = "Fetch payloads from a chain and output stats"
	app.Flags = flags
	app.Action = func(c *cli.Context) error {
		rpcURL := c.String("rpc-url")
		chainID := c.String("chain-id")
		genesisFilePath := c.String("genesis")
		sampleSize := c.Int("sample-size")

		var genesis *core.Genesis
		var err error
		if chainID != "" {
			genesisFile, err := os.Open(genesisFilePath)
			if err != nil {
				return err
			}
			defer func() { _ = genesisFile.Close() }()
			err = json.NewDecoder(genesisFile).Decode(&genesis)
			if err != nil {
				return err
			}
		} else {
			chainIDBig, ok := new(big.Int).SetString(chainID, 10)
			if !ok {
				return fmt.Errorf("invalid chain ID: %s", chainID)
			}

			genesis, err = core.LoadOPStackGenesis(chainIDBig.Uint64())
			if err != nil {
				return err
			}
		}

		client, err := ethclient.DialContext(c.Context, rpcURL)
		if err != nil {
			return err
		}

		// just do latest block for now
		latestBlock, err := client.BlockByNumber(c.Context, nil)
		if err != nil {
			return err
		}

		logger := log.NewLogger(os.Stdout, log.ReadCLIConfig(c))

		aggregateBlockStats := simulatorstats.NewStats()
		totalTxs := 0

		headerCache := make(map[common.Hash]*types.Header)

		allBlockStats := make([]*simulatorstats.Stats, sampleSize)

		for i := 0; i < sampleSize; i++ {
			logger.Info("Fetching block stats", "block", latestBlock.Number().String())

			blockStats, txStats, err := fetchBlockStats(logger, client, latestBlock, genesis, headerCache)
			if err != nil {
				return err
			}

			latestBlock, err = client.BlockByHash(c.Context, latestBlock.ParentHash())
			if err != nil {
				return err
			}

			aggregateBlockStats = aggregateBlockStats.Add(blockStats)
			allBlockStats[i] = blockStats
			totalTxs += len(txStats)
		}

		aggregateTxStats := aggregateBlockStats.Copy().Mul(1 / float64(totalTxs))
		aggregateBlockStats = aggregateBlockStats.Mul(1 / float64(sampleSize))

		blockVariance := simulatorstats.NewStats()
		// calculate std dev for each stat
		for i := 0; i < sampleSize; i++ {
			allBlockStats[i] = allBlockStats[i].Sub(aggregateBlockStats)
			allBlockStats[i] = allBlockStats[i].Pow(2)
			blockVariance = blockVariance.Add(allBlockStats[i])
		}

		blockVariance = blockVariance.Mul(1 / float64(sampleSize))
		_ = blockVariance.Pow(0.5)

		fmt.Printf("Aggregate block stats:\n%s\n\n", aggregateBlockStats)
		fmt.Printf("Aggregate tx stats:\n%s\n\n", aggregateTxStats)
		// fmt.Printf("Block std dev:\n%s\n\n", blockStdDev)
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
