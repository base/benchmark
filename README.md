<div align="center">
  <h1 style="font-size:32pt">Base Benchmark</h1>
  <a href="https://shields.io/"><img src="https://shields.io/badge/status-beta-yellow" alt="Status: Beta"></a>
  <a href="https://go.dev/"><img src="https://shields.io/badge/language-Go-00ADD8" alt="Language: Go"></a>
  <a href="https://github.com/base/benchmark/blob/main/LICENSE"><img src="https://shields.io/github/license/base/benchmark" alt="License"></a>
</div>

Base Benchmark is a performance testing framework for Ethereum execution clients. Compare client performance, identify bottlenecks, and ensure reliability before deployment.

## 🚀 Features

Base Benchmark provides comprehensive testing capabilities:

- **Performance Evaluation** - Test both block building and validation performance across execution clients
- **Comparative Analysis** - Measure client behavior across various inputs and workloads
- **Metric Collection** - Track critical metrics including submission times, latency, and throughput
- **Flexible Workloads** - Configure transaction patterns to match your specific needs
- **Visual Reports** - Generate interactive HTML dashboards of benchmark results

## 📋 Quick Start

```bash
# Build the application
make build

# Run the basic benchmark
./bin/base-bench run \
  --config ./configs/basic.yml \
  --root-dir ./data-dir \
  --reth-bin path_to_reth_bin \
  --geth-bin path_to_geth_bin \
  --output-dir ./output

# View the interactive dashboard
cd report/
npm i
npm run dev
```

## 🏗️ Architecture

### Benchmark Structure

Each benchmark consists of configurable tests with various input parameters:

```yaml
payloads:
  - name: Transfer only
    id: transfer-only
    type: transfer-only

benchmarks:
  - name: Test Performance
    description: Execution Speed
    variables:
      - type: payload
        value: transfer-only
      - type: node_type
        values:
          - reth
          - geth
      - type: num_blocks
        value: 20
```

This configuration runs a `transfer-only` transaction payload against both Geth and Reth clients for 20 blocks.

### Test Methodology

Each test executes a standardized workflow:

1. Initialize a sequencer/block builder with specified gas limits
2. Generate transactions and submit to the sequencer mempool
3. Record all payloads via `engine_forkChoiceUpdated` and `engine_getPayload`
4. Set up the validator node
5. Process payloads through `engine_newPayload`

This approach allows precise measurement of performance characteristics for both block production and validation.

## 🔧 Configuration

### Build

```bash
make build
ls ./bin/base-bench
```

### Available Flags

```
NAME:
   base-bench run - run benchmark

USAGE:
   base-bench run [command options]

OPTIONS:
   --config value                  Config Path ($BASE_BENCH_CONFIG)
   --root-dir value                Root Directory ($BASE_BENCH_ROOT_DIR)
   --output-dir value              Output Directory ($BASE_BENCH_OUTPUT_DIR)
   --tx-fuzz-bin value             Transaction Fuzzer path (default: "../tx-fuzz/cmd/livefuzzer/livefuzzer")

   # Reth Configuration
   --reth-bin value                Reth binary path (default: "reth")
   --reth-http-port value          HTTP port (default: 9545)
   --reth-auth-rpc-port value      Auth RPC port (default: 9551)
   --reth-metrics-port value       Metrics port (default: 9080)

   # Geth Configuration
   --geth-bin value                Geth binary path (default: "geth")
   --geth-http-port value          HTTP port (default: 8545)
   --geth-auth-rpc-port value      Auth RPC port (default: 8551)
   --geth-metrics-port value       Metrics port (default: 8080)

   # General Options
   --proxy-port value              Proxy port (default: 8546)
   --help, -h                      Show help (default: false)
```

## 📊 Example Reports

<div align="center">
  <p><i>Performance comparison between Geth and Reth clients</i></p>
</div>

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📜 License

This project is licensed under the [MIT License](LICENSE).
