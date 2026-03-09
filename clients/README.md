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

### build-base-reth-node.sh
Builds the base-reth-node and base-builder binaries from the base repository using Cargo.

**Default Configuration:**
- Repository: `https://github.com/base/base`
- Version: `main`
- Build tool: `cargo`

## Usage

### Using Makefile (Recommended)

```bash
# Build all binaries
make build-binaries

# Build only reth
make build-reth

# Build only geth
make build-geth

# Build base-reth-node and base-builder
make build-base-reth-node
```

### Direct Script Execution

```bash
# Build reth with defaults
cd clients
./build-reth.sh

# Build geth with defaults
./build-geth.sh

# Build base-reth-node and base-builder with defaults
./build-base-reth-node.sh
```

## Version Management

All client versions are managed in the `versions.env` file. This file contains the default repository URLs and versions for all supported clients. The build scripts automatically source this file if it exists.

### Customizing Repository and Version

You can override the default repository and version in several ways:

#### 1. Edit versions.env (Recommended)
Modify the `versions.env` file to change defaults for all builds:

```bash
# Edit versions.env to update default versions
OPTIMISM_VERSION="v0.2.0-beta.5"
GETH_VERSION="v1.13.0"
BASE_RETH_NODE_VERSION="your-commit-hash"
```

#### 2. Environment Variables
Override specific builds with environment variables:

```bash
# Build reth from a specific commit
OPTIMISM_REPO="https://github.com/ethereum-optimism/optimism/" OPTIMISM_VERSION="v0.1.0" ./build-reth.sh

# Build geth from a fork
GETH_REPO="https://github.com/your-fork/op-geth/" GETH_VERSION="your-branch" ./build-geth.sh

# Build base-reth-node and base-builder from a different commit
BASE_RETH_NODE_VERSION="your-commit-hash" ./build-base-reth-node.sh
```

### Available Environment Variables

#### For reth (build-reth.sh):
- `OPTIMISM_REPO`: Git repository URL (default: https://github.com/ethereum-optimism/optimism/)
- `OPTIMISM_VERSION`: Git branch, tag, or commit hash (default: develop)
- `BUILD_DIR`: Directory for source code (default: ./build)
- `OUTPUT_DIR`: Directory for built binaries (default: ../bin)

#### For geth (build-geth.sh):
- `GETH_REPO`: Git repository URL (default: https://github.com/ethereum-optimism/op-geth/)
- `GETH_VERSION`: Git branch, tag, or commit hash (default: optimism)
- `BUILD_DIR`: Directory for source code (default: ./build)
- `OUTPUT_DIR`: Directory for built binaries (default: ../bin)

#### For base-reth-node (build-base-reth-node.sh):
- `BASE_RETH_NODE_REPO`: Git repository URL (default: https://github.com/base/base)
- `BASE_RETH_NODE_VERSION`: Git branch, tag, or commit hash (default: main)
- `BUILD_DIR`: Directory for source code (default: ./build)
- `OUTPUT_DIR`: Directory for built binaries (default: ../bin)

## Prerequisites

### For reth:
- Rust and Cargo installed
- Git

### For geth:
- Go toolchain
- Git

### For base-reth-node:
- Rust and Cargo installed
- Git

## Output

Built binaries will be placed in the `bin/` directory at the project root:
- `bin/reth` - The reth binary
- `bin/geth` - The op-geth binary
- `bin/base-reth-node` - The base reth node binary
- `bin/base-builder` - The builder binary
