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

### build-builder.sh
Builds the builder binary from the op-rbuilder repository using Cargo.

**Default Configuration:**
- Repository: `https://github.com/base/op-rbuilder`
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

# Build only builder
make build-builder
```

### Direct Script Execution

```bash
# Build reth with defaults
cd clients
./build-reth.sh

# Build geth with defaults
./build-geth.sh

# Build builder with defaults
./build-builder.sh
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
BUILDER_VERSION="your-commit-hash"
```

#### 2. Environment Variables
Override specific builds with environment variables:

```bash
# Build reth from a specific commit
OPTIMISM_REPO="https://github.com/ethereum-optimism/optimism/" OPTIMISM_VERSION="v0.1.0" ./build-reth.sh

# Build geth from a fork
GETH_REPO="https://github.com/your-fork/op-geth/" GETH_VERSION="your-branch" ./build-geth.sh

# Build builder from a different commit
BUILDER_VERSION="main" ./build-builder.sh
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

#### For builder (build-builder.sh):
- `BUILDER_REPO`: Git repository URL (default: https://github.com/base/op-rbuilder)
- `BUILDER_VERSION`: Git branch, tag, or commit hash (default: main)
- `BUILD_DIR`: Directory for source code (default: ./build)
- `OUTPUT_DIR`: Directory for built binaries (default: ../bin)

## Prerequisites

### For reth:
- Rust and Cargo installed
- Git

### For geth:
- Go toolchain
- Git

### For builder:
- Rust and Cargo installed
- Git

## Output

Built binaries will be placed in the `bin/` directory at the project root:
- `bin/reth` - The reth binary
- `bin/geth` - The op-geth binary
- `bin/op-rbuilder` - The builder binary
