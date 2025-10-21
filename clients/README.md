# Client Build Scripts

This directory contains scripts to build client binaries for blockchain nodes.

## Available Scripts

### build-reth.sh
Builds the reth binary from the Paradigm reth repository using Cargo.

**Default Configuration:**
- Repository: `https://github.com/paradigmxyz/reth/`
- Version: `main`
- Build tool: `cargo`

### build-geth.sh
Builds the op-geth binary from the Ethereum Optimism op-geth repository using just.

**Default Configuration:**
- Repository: `https://github.com/ethereum-optimism/op-geth/`
- Version: `optimism`
- Build tool: `go run build/ci.go install`

### build-rbuilder.sh
Builds the op-rbuilder binary from the op-rbuilder repository using Cargo.

**Default Configuration:**
- Repository: `https://github.com/base/op-rbuilder.git`
- Version: `bc7e167a8d11362a78b9c30d59adcd8d2c7f9e84`
- Build tool: `cargo`

### build-rollup-boost.sh
Builds the rollup-boost binary from the Flashbots rollup-boost repository using Cargo.
Required for flashblocks dual-builder mode (production architecture).

**Default Configuration:**
- Repository: `https://github.com/flashbots/rollup-boost.git`
- Version: `08ebd3e75a8f4c7ebc12db13b042dee04e132c05`
- Build tool: `cargo`

**Note:** Rollup-boost is optional. Rbuilder can run in standalone mode without it.

## Usage

### Using Makefile (Recommended)

```bash
# Build all binaries
make build-binaries

# Build only reth
make build-reth

# Build only geth
make build-geth

# Build only op-rbuilder
make build-rbuilder

# Build only rollup-boost (for flashblocks dual-builder mode)
make build-rollup-boost
```

### Direct Script Execution

```bash
# Build reth with defaults
cd clients
./build-reth.sh

# Build geth with defaults
./build-geth.sh

# Build op-rbuilder with defaults
./build-rbuilder.sh

# Build rollup-boost with defaults (for flashblocks dual-builder mode)
./build-rollup-boost.sh
```

## Version Management

All client versions are managed in the `versions.env` file. This file contains the default repository URLs and versions for all supported clients. The build scripts automatically source this file if it exists.

### Customizing Repository and Version

You can override the default repository and version in several ways:

#### 1. Edit versions.env (Recommended)
Modify the `versions.env` file to change defaults for all builds:

```bash
# Edit versions.env to update default versions
RETH_VERSION="v0.2.0-beta.5"
GETH_VERSION="v1.13.0"
RBUILDER_VERSION="your-commit-hash"
```

#### 2. Environment Variables
Override specific builds with environment variables:

```bash
# Build reth from a specific commit
RETH_REPO="https://github.com/paradigmxyz/reth/" RETH_VERSION="v0.1.0" ./build-reth.sh

# Build geth from a fork
GETH_REPO="https://github.com/your-fork/op-geth/" GETH_VERSION="your-branch" ./build-geth.sh

# Build op-rbuilder from a different commit
RBUILDER_VERSION="main" ./build-rbuilder.sh
```

### Available Environment Variables

#### For reth (build-reth.sh):
- `RETH_REPO`: Git repository URL (default: https://github.com/paradigmxyz/reth/)
- `RETH_VERSION`: Git branch, tag, or commit hash (default: main)
- `BUILD_DIR`: Directory for source code (default: ./build)
- `OUTPUT_DIR`: Directory for built binaries (default: ../bin)

#### For geth (build-geth.sh):
- `GETH_REPO`: Git repository URL (default: https://github.com/ethereum-optimism/op-geth/)
- `GETH_VERSION`: Git branch, tag, or commit hash (default: optimism)
- `BUILD_DIR`: Directory for source code (default: ./build)
- `OUTPUT_DIR`: Directory for built binaries (default: ../bin)

#### For op-rbuilder (build-rbuilder.sh):
- `RBUILDER_REPO`: Git repository URL (default: https://github.com/base/op-rbuilder.git)
- `RBUILDER_VERSION`: Git branch, tag, or commit hash (default: bc7e167a8d11362a78b9c30d59adcd8d2c7f9e84)
- `BUILD_DIR`: Directory for source code (default: ./build)
- `OUTPUT_DIR`: Directory for built binaries (default: ../bin)

#### For rollup-boost (build-rollup-boost.sh):
- `ROLLUP_BOOST_REPO`: Git repository URL (default: https://github.com/flashbots/rollup-boost.git)
- `ROLLUP_BOOST_VERSION`: Git branch, tag, or commit hash (default: 08ebd3e75a8f4c7ebc12db13b042dee04e132c05)
- `BUILD_DIR`: Directory for source code (default: ./build)
- `OUTPUT_DIR`: Directory for built binaries (default: ../bin)

## Prerequisites

### For reth:
- Rust and Cargo installed
- Git

### For geth:
- Go toolchain
- Git

### For op-rbuilder:
- Rust and Cargo installed
- Git

### For rollup-boost:
- Rust and Cargo installed
- Git
- `libssl-dev` and `pkg-config` (on Linux)

## Output

Built binaries will be placed in the `bin/` directory at the project root:
- `bin/reth` - The reth binary
- `bin/geth` - The op-geth binary
- `bin/op-rbuilder` - The op-rbuilder binary
- `bin/rollup-boost` - The rollup-boost binary (for flashblocks dual-builder mode)

## Flashblocks Dual-Builder Mode

To run benchmarks with the full flashblocks architecture:

1. Build the required binaries:
   ```bash
   make build-rbuilder
   make build-rollup-boost
   make build-reth  # or build-geth for fallback
   ```

2. Run benchmark with dual-builder flags:
   ```bash
   ./bin/base-bench run \
     --config ./configs/examples/flashblocks.yml \
     --rbuilder-bin ./bin/op-rbuilder \
     --flashblocks-fallback reth \
     --rollup-boost-bin ./bin/rollup-boost \
     --output-dir ./output
   ```

See [configs/examples/flashblocks.yml](../configs/examples/flashblocks.yml) for complete examples. 