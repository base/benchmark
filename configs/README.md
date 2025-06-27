# Base Benchmark Configurations

This directory contains benchmark configurations for testing various aspects of Ethereum execution client performance. The configurations are organized into two categories:

- **[examples/](./examples/)** - Development and testing configurations with specific focused workloads
- **[public/](./public/)** - Production-ready benchmarks for standardized testing and comparison

## 📁 Examples Configurations

| Configuration                                      | Type           | Purpose                                         | Gas Limits |
| -------------------------------------------------- | -------------- | ----------------------------------------------- | ---------- |
| [📄 sload.yml](./examples/sload.yml)               | Storage        | Tests SLOAD operation performance               | 1B         |
| [📄 sstore.yml](./examples/sstore.yml)             | Storage        | Tests SSTORE operation performance              | 1B         |
| [📄 ecadd.yml](./examples/ecadd.yml)               | Precompile     | Tests EC addition precompile (0x06)             | 1B         |
| [📄 ecmul.yml](./examples/ecmul.yml)               | Precompile     | Tests EC multiplication precompile (0x07)       | 1B         |
| [📄 ecpairing.yml](./examples/ecpairing.yml)       | Precompile     | Tests EC pairing precompile (0x08) under stress | 1B         |
| [📄 contract.yml](./examples/contract.yml)         | Precompile     | Basic EC pairing precompile test                | 1B         |
| [📄 erc20.yml](./examples/erc20.yml)               | Contract       | Tests ERC-20 token transfer performance         | 1B         |
| [📄 simulator.yml](./examples/simulator.yml)       | Simulation     | Comprehensive workload with mixed operations    | 90M        |
| [📄 snapshot.yml](./examples/snapshot.yml)         | Infrastructure | Tests snapshot creation and loading             | 15M-90M    |
| [📄 tx-fuzz-geth.yml](./examples/tx-fuzz-geth.yml) | Stress Test    | Randomized transaction pattern testing          | Default    |

## 📁 Public Configurations

| Configuration                                                      | Purpose                       | Gas Limits | Clients    |
| ------------------------------------------------------------------ | ----------------------------- | ---------- | ---------- |
| [📄 basic.yml](./public/basic.yml)                                 | Baseline transfer performance | 15M-90M    | Geth, Reth |
| [📄 mainnet-cross-section.yml](./public/mainnet-cross-section.yml) | Base mainnet simulation       | 25M-100M   | Geth       |
| [📄 public-benchmark.yml](./public/public-benchmark.yml)           | Standard benchmark suite      | 15M-1005M  | Geth, Reth |
| [📄 proof-program.yml](./public/proof-program.yml)                 | Fault proof program testing   | 15M        | Geth       |

## 🚀 Usage Examples

### Run a specific example

```bash
./bin/base-bench run \
  --config ./configs/examples/erc20.yml \
  --root-dir ./data-dir \
  --reth-bin path_to_reth_bin \
  --geth-bin path_to_geth_bin \
  --output-dir ./output
```

### Run public benchmark

```bash
./bin/base-bench run \
  --config ./configs/public/mainnet-cross-section.yml \
  --root-dir ./data-dir \
  --reth-bin path_to_reth_bin \
  --geth-bin path_to_geth_bin \
  --output-dir ./output
```

### Run multiple configurations

```bash
# Run all storage operation tests
for config in ./configs/examples/sload.yml ./configs/examples/sstore.yml; do
  ./bin/base-bench run --config "$config" --root-dir ./data-dir --output-dir ./output
done
```

## 📊 Configuration Structure

Each benchmark configuration follows this structure:

```yaml
payloads:
  - name: "Descriptive Name"
    id: unique-identifier
    type: transfer-only|contract|simulator|tx-fuzz
    # ... payload-specific parameters

benchmarks:
  - name: "Benchmark Name"
    description: "What this benchmark tests"
    variables:
      - type: payload|node_type|num_blocks|gas_limit
        value: single-value
        values: [array, of, values] # for matrix testing
```

## 🎯 Choosing the Right Configuration

- **Development/Testing**: Use `examples/` configurations for focused testing
- **Production Validation**: Use `public/` configurations for comprehensive testing
- **Performance Regression**: Use `basic.yml` for quick CI/CD checks
- **Mainnet Readiness**: Use `mainnet-cross-section.yml` for full validation
- **Specific Features**: Choose targeted configs (storage, precompiles, etc.)

## 📈 Performance Metrics

All configurations track key metrics:

- **Sequencer Metrics**: Block building time, payload generation latency
- **Validator Metrics**: Block validation time, payload processing latency
- **Resource Usage**: CPU, memory, disk I/O
- **Throughput**: Transactions per second, gas per second

For detailed metric analysis, view the interactive dashboard after running benchmarks.
