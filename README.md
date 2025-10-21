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

[Install Forge](https://book.getfoundry.sh/getting-started/installation)

Recursively clone github submodules:

```bash
git submodule update --init --recursive
```

```bash
# Build the application
make build

# Build the binaries, geth, reth, rbuilder
make build-binaries

# Run the basic benchmark
./bin/base-bench run \
  --config ./configs/public/basic.yml \
  --root-dir ./data-dir \
  --reth-bin path_to_reth_bin \
  --geth-bin path_to_geth_bin \
  --output-dir ./output

# View the interactive dashboard
cd report/
npm i
npm run dev
```

## 📋 Available Benchmarks

Explore the comprehensive collection of benchmark configurations:

**[📁 Configuration Guide](configs/README.md)** - Detailed documentation of all available benchmark configurations

- **[examples/](configs/examples/)** - Development and testing configurations for specific workloads
- **[public/](configs/public/)** - Production-ready benchmarks for standardized testing

Choose from storage operations, precompile tests, token workloads, mainnet simulations, and more.

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

### Flashblocks Support

The benchmark system supports **flashblocks** - Base's sub-200ms transaction preconfirmation feature via the `rbuilder` client.

**Two Modes:**

1. **Simple mode (standalone)**: Just rbuilder for testing
   ```yaml
   benchmarks:
     - variables:
         - type: node_type
           value: rbuilder
   ```

2. **Dual-builder mode (production architecture)**: Fallback builder + rbuilder + optional rollup-boost
   ```bash
   ./bin/base-bench run \
     --config ./configs/examples/flashblocks.yml \
     --root-dir ./data-dir \
     --reth-bin ./bin/op-reth \
     --geth-bin ./bin/geth \
     --rbuilder-bin ./bin/op-rbuilder \
     --flashblocks-fallback reth \
     --rollup-boost-bin ./bin/rollup-boost \
     --output-dir ./output
   ```

**How it works (dual-builder mode):**
- **Fallback builder** (geth/reth): Produces final 2s canonical blocks
- **Rbuilder** (primary): Produces flashblocks every 200ms (10 per 2s block)
- **Rollup-boost** (optional): Coordinates between the two builders

**Configurable flashblock interval:**
```yaml
- type: flashblock_interval
  values:
    - 100   # 100ms (aggressive)
    - 200   # 200ms (Base default)
    - 500   # 500ms (conservative)
```

See [configs/examples/flashblocks.yml](configs/examples/flashblocks.yml) for complete examples.

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

   # Geth Configuration
   --geth-bin value                Geth binary path (default: "geth")

   # Rbuilder (Flashblocks) Configuration
   --rbuilder-bin value            Rbuilder binary path (default: "rbuilder")
   --flashblocks-fallback value    Fallback client for dual-builder mode: geth or reth (default: "reth")
   --rollup-boost-bin value        Rollup-boost coordinator binary for dual-builder mode

   # General Options
   --proxy-port value              Proxy port (default: 8546)
   --help, -h                      Show help (default: false)
```

**Note**: Without `--flashblocks-fallback`, rbuilder runs in simple standalone mode. Set `--flashblocks-fallback` to enable production dual-builder architecture.

### Client-Specific Arguments

You can customize client behavior using the `client_args` variable in YAML configs:

```yaml
benchmarks:
  - variables:
      - type: client_args
        value:
          geth: "--verbosity 4 --txpool.globalslots 20000000"
          reth: "--txpool.pending-max-count 200000000 -vvvv"
          rbuilder: "--txpool.pending-max-count 200000000"
```

This allows testing different client configurations without code changes.

## 📊 Example Reports

<div align="center">
  <p><i>Performance comparison between Geth and Reth clients</i></p>
</div>

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📜 License

This project is licensed under the [MIT License](LICENSE).
