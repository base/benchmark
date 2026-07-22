package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/core"
)

// dump-genesis prints the OP-Stack genesis for a given chain ID as JSON.
// Usage: dump-genesis <chain-id>
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: dump-genesis <chain-id>")
		os.Exit(1)
	}
	chainID, err := strconv.ParseUint(os.Args[1], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid chain id: %v\n", err)
		os.Exit(1)
	}
	genesis, err := core.LoadOPStackGenesis(chainID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load genesis: %v\n", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(genesis); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode genesis: %v\n", err)
		os.Exit(1)
	}
}
