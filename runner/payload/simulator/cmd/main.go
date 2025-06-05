package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "payload-simulator"
	app.Usage = "Fetch payloads from a chain and output stats"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "rpc-url",
			Usage:    "RPC URL of the chain to fetch payloads from",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "block-range",
			Usage:    "Block range to fetch payloads from",
			Required: false,
		},
		&cli.StringFlag{
			Name:  "output",
			Usage: "Path to the output JSON file",
			Value: "stats.json",
		},
		&cli.IntFlag{
			Name:  "sample-size",
			Usage: "Number of payloads to sample",
			Value: 100,
		},
		&cli.StringFlag{
			Name:  "genesis",
			Usage: "Genesis JSON file",
			Value: "genesis.json",
		},
	}
	app.Action = func(c *cli.Context) error {
		rpcURL := c.String("rpc-url")
		genesisFilePath := c.String("genesis")
		// blockRange := c.String("block-range")
		// output := c.String("output")
		// sampleSize := c.Int("sample-size")

		var genesis *core.Genesis
		genesisFile, err := os.Open(genesisFilePath)
		if err != nil {
			return err
		}
		defer genesisFile.Close()
		err = json.NewDecoder(genesisFile).Decode(&genesis)
		if err != nil {
			return err
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

		_, err = fetchBlockStats(client, latestBlock, genesis)
		if err != nil {
			return err
		}

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
