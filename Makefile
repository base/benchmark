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

.PHONY: contracts
contracts:
	forge build --force

.PHONY: build
build:
	forge build --force && env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) CGO_ENABLED=0 go build -v $(LDFLAGS) -o ./bin/base-bench ./benchmark/cmd

.PHONY: clean
clean:
	rm bin/base-bench

.PHONY: test
test:
	go test -v ./...
