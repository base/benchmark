![Base](.github/assets/logo.png)

# Base Benchmark

Base Benchmark is a performance testing framework for Ethereum execution clients. Compare client performance, identify bottlenecks, and ensure reliability before deployment.

<!-- Badge row 1 - status -->

[![GitHub contributors](https://img.shields.io/github/contributors/base/benchmark)](https://github.com/base/benchmark/graphs/contributors)
[![GitHub commit activity](https://img.shields.io/github/commit-activity/w/base/benchmark)](https://github.com/base/benchmark/graphs/contributors)
[![GitHub Stars](https://img.shields.io/github/stars/base/benchmark.svg)](https://github.com/base/benchmark/stargazers)
![GitHub repo size](https://img.shields.io/github/repo-size/base/benchmark)
[![GitHub](https://img.shields.io/github/license/base/benchmark?color=blue)](https://github.com/base/benchmark/blob/main/LICENSE)

<!-- Badge row 2 - links and profiles -->

[![Website base.org](https://img.shields.io/website-up-down-green-red/https/base.org.svg)](https://base.org)
[![Blog](https://img.shields.io/badge/blog-up-green)](https://base.mirror.xyz/)
[![Docs](https://img.shields.io/badge/docs-up-green)](https://docs.base.org/)
[![Discord](https://img.shields.io/discord/1067165013397213286?label=discord)](https://base.org/discord)
[![Twitter Base](https://img.shields.io/twitter/follow/Base?style=social)](https://twitter.com/Base)

<!-- Badge row 3 - detailed status -->

[![GitHub pull requests by-label](https://img.shields.io/github/issues-pr-raw/base/benchmark)](https://github.com/base/benchmark/pulls)
[![GitHub Issues](https://img.shields.io/github/issues-raw/base/benchmark.svg)](https://github.com/base/benchmark/issues)

## Results

Public results are available at the following links:

| Network      | Link                                                                   |
| ------------ | ---------------------------------------------------------------------- |
| Devnet       | [https://base.github.io/benchmark/](https://base.github.io/benchmark/) |
| Base Sepolia | Coming soon                                                            |
| Base Mainnet | Coming soon                                                            |

## Features

- **Performance Evaluation:** Test both block building and validation performance across execution clients (Geth, Reth, and more)
- **Comparative Analysis:** Measure client behavior across various inputs and workloads
- **Metric Collection:** Track critical metrics including submission times, latency, and throughput
- **Flexible Workloads:** Configure transaction patterns to match your specific needs
- **Interactive Dashboard:** Generate beautiful HTML reports with charts and run comparisons
- **Import & Merge:** Combine benchmark results from multiple machines with flexible tagging

## Repository Structure

```
.
├── Makefile              # Build and development tasks
├── go.mod                # Go module dependencies
├── benchmark/            # CLI application
│   ├── cmd/              # Main entry point
│   ├── config/           # Configuration types
│   └── flags/            # CLI flags
├── runner/               # Core benchmarking logic
│   ├── benchmark/        # Benchmark execution
│   ├── clients/          # Client integrations (Geth, Reth)
│   ├── importer/         # Run import functionality
│   ├── network/          # Network setup and management
│   └── payload/          # Transaction payload generation
├── configs/              # Benchmark configurations
│   ├── examples/         # Development and testing configs
│   └── public/           # Production-ready benchmarks
├── contracts/            # Smart contracts for testing
│   └── src/              # Solidity source files
├── report/               # Interactive dashboard
│   └── src/              # React TypeScript application
└── clients/              # Client build scripts
```

## Prerequisites

- **Go:** Version 1.21 or later. Install from [go.dev](https://go.dev/dl/)
- **Foundry:** For smart contract compilation. See [installation guide](https://book.getfoundry.sh/getting-started/installation)
- **Node.js:** Version 18+ for the interactive dashboard. Install from [nodejs.org](https://nodejs.org/)

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/base/benchmark.git
cd benchmark
git submodule update --init --recursive
```

### 2. Build the Application

```bash
make build
```

The binary will be located at `bin/benchmark`.

### 3. Build Client Binaries (Optional)

To build Geth and Reth from source:

```bash
make build-binaries
```

Alternatively, you can specify paths to pre-built binaries when running benchmarks.

### 4. Run Your First Benchmark

```bash
./bin/base-bench run \
  --config ./configs/public/basic.yml \
  --root-dir ./data-dir \
  --output-dir ./output
```

To see available options:

```bash
./bin/base-bench run --help
```

### 5. View Results in the Interactive Dashboard

```bash
cd report/
npm install
npm run dev
```

Open your browser to the URL shown (typically `http://localhost:5173`).

### Visualize snapshot-backed devnet benchmarks

Results produced by `base-bench snapshot` in the `base/base` repository can be
imported directly. The importer preserves sequencer and validator canonical
block metrics, derives actual gas/s and transactions/s at the configured block
cadence, and copies the load-generator summary alongside each run.

```bash
./scripts/import-snapshot-benchmarks.py \
  --client-version "$(git -C ../base rev-parse --short HEAD)" \
  /path/to/existing-account-2s.json \
  /path/to/existing-account-200ms.json \
  /path/to/fresh-account-2s.json \
  /path/to/fresh-account-200ms.json

make build-server
./bin/report-server --local-dir ./output

# In another shell:
cd report
yarn install
VITE_DATA_SOURCE=api \
  VITE_API_BASE_URL=http://localhost:8080/ \
  VITE_ALLOWED_HOSTS=eagle,localhost \
  yarn dev --host 0.0.0.0
```

Open
[`http://localhost:3000/#/run-comparison/snapshot-throughput`](http://localhost:3000/#/run-comparison/snapshot-throughput).
Use **Show Line Per → Block Time Milliseconds**, then select the workload and
node role to compare 2s and 200ms blocks. The workload is exposed through the
standard **Transaction Payload** filter. Generated data lives in the ignored
`output/` directory, so benchmark artifacts do not need to be committed.

Each imported result owns one directory containing its role metrics,
load-test artifact, and a one-run `metadata.json`. The importer writes metadata
last as the commit signal. The report server discovers these per-run files and
assembles `/output/metadata.json`; producers never update a shared metadata file.

The imported report covers canonical gas/block, gas/s, transactions/block,
transactions/s, and load-test totals. It does not yet collect host CPU, memory,
disk I/O, txpool depth, or payload deadline metrics.

## Available Benchmarks

Explore the comprehensive collection of benchmark configurations:

**[📁 Configuration Guide](configs/README.md)** - Detailed documentation of all available benchmark configurations

- **[examples/](configs/examples/)** - Development and testing configurations for specific workloads
- **[public/](configs/public/)** - Production-ready benchmarks for standardized testing

Choose from storage operations, precompile tests, token workloads, mainnet simulations, and more.

## Tools

### Payload Simulator

The **[Payload Simulator](runner/payload/simulator/README.md)** analyzes real-world block execution characteristics by fetching blocks from live chains and computing statistics about:

- Account and storage operations (reads, writes, creates, deletes)
- Opcode usage patterns (EXP, KECCAK256, etc.)
- Precompile calls (ecrecover, bn256, BLS12-381, etc.)

Use it to generate realistic benchmark configurations based on actual mainnet data:

```bash
go build -o bin/payload-simulator ./runner/payload/simulator/cmd

# RPC must support debug_executionWitness
./bin/payload-simulator \
  --rpc-url <your-rpc-url> \
  --sample-size 100
```

## Architecture

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

Benchmarks run both phases by default. Set `roles: [sequencer]` on a benchmark definition to run only the sequencer/block-building phase, which is useful for snapshot startup and load-test coverage that does not need validator payload replay.

## Configuration

### Available Flags

```
NAME:
   benchmark run - run benchmark

USAGE:
   benchmark run [command options]

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

## Managing Test Runs

### Understanding Runs and Suites

When you view benchmark results in the interactive dashboard, you can switch between different test runs using the run switcher:

<div align="center">
  <img src=".github/assets/run-switcher.png" alt="Run Switcher" width="600">
</div>

### Creating Test Runs

Running benchmarks adds a new suite by default:

```bash
./bin/base-bench run --config ./configs/public/basic.yml
```

Each execution creates a new suite entry in the run list, allowing you to track performance over time or across different configurations.

### Combining Multiple Runs

Use `import-runs` to merge benchmark results from multiple machines or configurations:

```bash
./bin/base-bench import-runs \
  --output-dir ./output \
  ./results-from-server-1/metadata.json
```

**Two import strategies:**

1. **Add to latest suite with tags** - Merge imported runs into your most recent suite, using tags to differentiate:

   ```bash
   # Add imported runs to the last suite with tags for differentiation
   ./bin/base-bench import-runs \
     --src-tag "instance=server-lg" \
     --dest-tag "instance=server-md" \
     --output-dir ./output \
     ./results-from-server-1/metadata.json

   # --src-tag fills missing tags on existing runs (won't overwrite)
   # --dest-tag applies to the imported runs
   # Useful for comparing hardware configurations within the same test run
   ```

2. **Create new separate suite** - Add imported runs as an independent suite in the list:

   ```bash
   # Interactive mode (recommended) - prompts you to choose strategy and configure tags
   ./bin/base-bench import-runs \
     --output-dir ./output \
     ./results-from-server-1/metadata.json

   # Creates a new entry differentiated by BenchmarkRun ID
   # Useful for tracking performance across different code versions or time periods
   ```

**Interactive Mode:** Without specifying tags, the tool enters interactive mode and guides you through:

- Choosing between adding to last suite or creating new suite
- Configuring appropriate tags if needed
- Confirming the import operation

This flexibility lets you organize benchmarks by hardware type, client version, or any dimension relevant to your analysis.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute to this project.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

**Built with ❤️ by [Base](https://base.org)**
