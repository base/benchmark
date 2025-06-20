build: build-abi build-bin build-abi-go

build-abi: src/**
	mkdir -p abi
	forge inspect src/Simulator.sol:Simulator abi --json > abi/Simulator.json

build-bin: src/**
	forge build --extra-output-files bin --force

build-abi-go: src/ build-abi
	go run github.com/ethereum/go-ethereum/cmd/abigen --abi abi/Simulator.json --pkg abi --type Simulator --out ../runner/payload/simulator/abi/Simulator.go --bin ./out/Simulator.sol/Simulator.bin