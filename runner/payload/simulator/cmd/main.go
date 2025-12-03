package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"sort"
	"sync"

	"github.com/base/base-bench/runner/payload/simulator/simulatorstats"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
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
	&cli.IntFlag{
		Name:  "sample-range",
		Usage: "Range of blocks to sample from (defaults to sample-size for consecutive blocks). Set to a larger value to pick random blocks over a wider timeframe.",
		Value: 0,
	},
	&cli.IntFlag{
		Name:  "num-workers",
		Usage: "Number of parallel workers for fetching and processing blocks",
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
	&cli.StringFlag{
		Name:  "client",
		Usage: "Client type for fetching preimages: 'geth' uses debug_dbGet, 'reth' uses debug_executionWitness",
		Value: "reth",
	},
}

func init() {
	flags = append(flags, oplog.CLIFlags("SIM")...)
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
		sampleRange := c.Int("sample-range")
		numWorkers := c.Int("num-workers")
		clientType := c.String("client")

		// Validate client type
		if clientType != "geth" && clientType != "reth" {
			return fmt.Errorf("invalid client type: %s (must be 'geth' or 'reth')", clientType)
		}

		// Default sample-range to sample-size (consecutive blocks)
		if sampleRange <= 0 {
			sampleRange = sampleSize
		}
		if sampleRange < sampleSize {
			return fmt.Errorf("sample-range (%d) must be >= sample-size (%d)", sampleRange, sampleSize)
		}
		if numWorkers < 1 {
			numWorkers = 1
		}

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

		latestBlock, err := client.BlockByNumber(c.Context, nil)
		if err != nil {
			return err
		}
		latestBlockNum := latestBlock.NumberU64()

		logger := oplog.NewLogger(os.Stdout, oplog.ReadCLIConfig(c))

		// Select which block numbers to sample
		blockNumbers := selectBlockNumbers(latestBlockNum-100, sampleSize, sampleRange)

		logger.Info("Starting parallel block processing", "blocks", len(blockNumbers), "workers", numWorkers, "client", clientType)

		// Process blocks in parallel using worker pool
		results, err := processBlocksParallel(c.Context, logger, client, genesis, blockNumbers, numWorkers, clientType)
		if err != nil {
			return err
		}

		// Aggregate results
		aggregateBlockStats := simulatorstats.NewStats()
		totalTxs := 0
		allBlockStats := make([]*simulatorstats.Stats, len(results))

		for i, result := range results {
			aggregateBlockStats = aggregateBlockStats.Add(result.blockStats)
			allBlockStats[i] = result.blockStats
			totalTxs += result.txCount
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

// blockResult holds the result of processing a single block.
type blockResult struct {
	index      int
	blockStats *simulatorstats.Stats
	txCount    int
	err        error
}

// processBlocksParallel fetches and processes blocks in parallel using a worker pool.
func processBlocksParallel(
	ctx context.Context,
	logger log.Logger,
	client *ethclient.Client,
	genesis *core.Genesis,
	blockNumbers []uint64,
	numWorkers int,
	clientType string,
) ([]blockResult, error) {
	// Channels for work distribution and result collection
	jobs := make(chan struct {
		index    int
		blockNum uint64
	}, len(blockNumbers))
	results := make(chan blockResult, len(blockNumbers))

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Each worker has its own header cache to avoid contention
			headerCache := make(map[common.Hash]*types.Header)

			for job := range jobs {
				block, err := client.BlockByNumber(ctx, big.NewInt(int64(job.blockNum)))
				if err != nil {
					results <- blockResult{index: job.index, err: fmt.Errorf("failed to fetch block %d: %w", job.blockNum, err)}
					continue
				}

				logger.Info("Processing block", "block", block.Number().String(), "index", job.index+1, "total", len(blockNumbers))

				// Select fetch function based on client type
				var blockStats *simulatorstats.Stats
				var txStats []*simulatorstats.Stats
				switch clientType {
				case "geth":
					blockStats, txStats, err = fetchBlockStatsGeth(logger, client, block, genesis, headerCache)
				case "reth":
					blockStats, txStats, err = fetchBlockStatsReth(logger, client, block, genesis, headerCache)
				}
				if err != nil {
					results <- blockResult{index: job.index, err: fmt.Errorf("failed to process block %d: %w", job.blockNum, err)}
					continue
				}

				results <- blockResult{
					index:      job.index,
					blockStats: blockStats,
					txCount:    len(txStats),
				}
			}
		}()
	}

	// Send jobs to workers
	for i, blockNum := range blockNumbers {
		jobs <- struct {
			index    int
			blockNum uint64
		}{index: i, blockNum: blockNum}
	}
	close(jobs)

	// Wait for all workers to finish and close results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	collected := make([]blockResult, len(blockNumbers))
	for result := range results {
		if result.err != nil {
			return nil, result.err
		}
		collected[result.index] = result
	}

	return collected, nil
}

// selectBlockNumbers returns a slice of block numbers to sample.
// If sampleRange equals sampleSize, returns consecutive blocks ending at latestBlockNum.
// Otherwise, randomly selects sampleSize blocks from the range [latestBlockNum-sampleRange+1, latestBlockNum].
func selectBlockNumbers(latestBlockNum uint64, sampleSize, sampleRange int) []uint64 {
	// Calculate the starting block number (ensure we don't go below 1)
	startBlock := uint64(1)
	if latestBlockNum > uint64(sampleRange-1) {
		startBlock = latestBlockNum - uint64(sampleRange-1)
	}

	// If range equals size, return consecutive blocks (original behavior)
	if sampleRange == sampleSize {
		blocks := make([]uint64, sampleSize)
		for i := 0; i < sampleSize; i++ {
			blocks[i] = latestBlockNum - uint64(i)
		}
		return blocks
	}

	// Randomly select sampleSize unique blocks from the range
	availableBlocks := int(latestBlockNum - startBlock + 1)
	if availableBlocks < sampleSize {
		// Not enough blocks available, use all of them
		blocks := make([]uint64, availableBlocks)
		for i := 0; i < availableBlocks; i++ {
			blocks[i] = startBlock + uint64(i)
		}
		return blocks
	}

	// Use reservoir sampling approach: generate random indices
	selectedIndices := make(map[int]struct{}, sampleSize)
	for len(selectedIndices) < sampleSize {
		idx := rand.Intn(availableBlocks)
		selectedIndices[idx] = struct{}{}
	}

	// Convert indices to block numbers
	blocks := make([]uint64, 0, sampleSize)
	for idx := range selectedIndices {
		blocks = append(blocks, startBlock+uint64(idx))
	}

	// Sort blocks in descending order (newest first) for consistent behavior
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i] > blocks[j]
	})

	return blocks
}
