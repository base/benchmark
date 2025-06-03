package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	// calculate hash of 0E345561ed8251FC66eE99DFd34f803cB2c30023 with NewLegacyKeccak256
	address := common.FromHex("0xC80A949A0c6799A385532A0ce471E1306A3c144D")
	hash := crypto.Keccak256Hash([]byte(address))
	fmt.Printf("Hash of address %x: %s\n", address, hash.Hex())

	// read from addrs.txt and do this for each address

	file, err := os.Open("addrs.txt")
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		addr := scanner.Text()
		if len(addr) == 0 {
			continue
		}
		address := common.FromHex(addr)
		hash := crypto.Keccak256Hash([]byte(address))
		fmt.Printf("Hash of address %s: %s\n", addr, hash.Hex()[15:])
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
}
