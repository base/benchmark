GITCOMMIT ?= $(shell git rev-parse HEAD)
GITDATE ?= $(shell git show -s --format='%ct')
# Find the github tag that points to this commit. If none are found, set the version string to "untagged"
# Prioritizes release tag, if one exists, over tags suffixed with "-rc"
VERSION ?= $(shell tags=$$(git tag --points-at $(GITCOMMIT) | grep '^op-batcher/' | sed 's/op-batcher\///' | sort -V); \
             preferred_tag=$$(echo "$$tags" | grep -v -- '-rc' | tail -n 1); \
             if [ -z "$$preferred_tag" ]; then \
                 if [ -z "$$tags" ]; then \
                     echo "untagged"; \
                 else \
                     echo "$$tags" | tail -n 1; \
                 fi \
             else \
                 echo $$preferred_tag; \
             fi)

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.Version=$(VERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

# Include .env file if it exists
-include .env

# first so that make defaults to building the benchmark
.PHONY: build
build:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) CGO_ENABLED=0 go build -v $(LDFLAGS) -o ./bin/base-bench ./benchmark/cmd

.PHONY: contracts
contracts:
	make -C contracts

.PHONY: clean
clean:
	rm bin/base-bench

.PHONY: test
test:
	go test -v ./...

.PHONY: build-reth
build-reth:
ifeq ($(OS),Windows_NT)
	cd clients && powershell -ExecutionPolicy Bypass -File build-reth.ps1
else
	cd clients && ./build-reth.sh
endif

.PHONY: build-geth
build-geth:
ifeq ($(OS),Windows_NT)
	cd clients && powershell -ExecutionPolicy Bypass -File build-geth.ps1
else
	cd clients && ./build-geth.sh
endif

.PHONY: build-rbuilder
build-rbuilder:
ifeq ($(OS),Windows_NT)
	cd clients && powershell -ExecutionPolicy Bypass -File build-rbuilder.ps1
else
	cd clients && ./build-rbuilder.sh
endif

.PHONY: build-binaries
build-binaries: build-reth build-geth build-rbuilder

.PHONY: build-frontend
build-frontend:
	cd report && npm run build

.PHONY: run-frontend
run-frontend:
ifeq ($(OS),Windows_NT)
	cd report && npm run dev
else
	cd report && set -a && [ -f ../.env ] && . ../.env && set +a && npm run dev
endif
