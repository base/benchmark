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
	cd clients && ./build-reth.sh

.PHONY: build-geth
build-geth:
	cd clients && ./build-geth.sh

.PHONY: build-rbuilder
build-rbuilder:
	cd clients && ./build-rbuilder.sh

.PHONY: build-binaries
build-binaries: build-reth build-geth build-rbuilder

.PHONY: build-backend
build-backend:
	cd report/backend && env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) CGO_ENABLED=0 go build -v $(LDFLAGS) -o ../../bin/base-bench-api cmd/main.go

.PHONY: build-frontend
build-frontend:
	cd report && yarn build

.PHONY: run-backend
run-backend:
	./bin/base-bench-api --s3-bucket ${BASE_BENCH_API_S3_BUCKET}

.PHONY: run-frontend
run-frontend:
	cd report && yarn dev

.PHONY: run-backfill
run-backfill:
	./bin/base-bench backfill-benchmark-run-id --s3-bucket ${BASE_BENCH_API_S3_BUCKET} metadata.json
