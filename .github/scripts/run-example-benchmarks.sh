#!/bin/bash
set -eo pipefail

BENCHMARK_CONFIGS=(
    configs/examples/contract.yml
    configs/examples/ecadd.yml
    configs/examples/ecmul.yml
    configs/examples/ecpairing.yml
    configs/examples/erc20.yml
    configs/examples/simulator.yml
    configs/examples/sload.yml
    # configs/examples/snapshot.yml
    configs/examples/sstore.yml
    # configs/examples/tx-fuzz-geth.yml
)

TEMP_DIR=$1

for config in "${BENCHMARK_CONFIGS[@]}"; do
    echo "Running $config"
    # each config will add on to the same output directory
    go run benchmark/cmd/main.go \
        --log.level debug \
        run \
        --config $config \
        --root-dir $TEMP_DIR/data-dir \
        --output-dir $TEMP_DIR/output \
        --reth-bin $TEMP_DIR/bin/reth \
        --geth-bin $TEMP_DIR/bin/geth
done