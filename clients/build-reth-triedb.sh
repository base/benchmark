#!/bin/bash

set -e

# Source versions if available, otherwise use defaults
if [ -f "versions.env" ]; then
    source versions.env
else
    # Default values
    RETH_TRIEDB_REPO="${RETH_TRIEDB_REPO:-https://github.com/base/reth-triedb/pull/17}"
    RETH_TRIEDB_VERSION="${RETH_TRIEDB_VERSION:-main}"
    BUILD_DIR="${BUILD_DIR:-./build}"
    OUTPUT_DIR="${OUTPUT_DIR:-../bin}"
fi

echo "Building reth-triedb binary..."
echo "Repository: $RETH_TRIEDB_REPO"
echo "Version/Commit: $RETH_TRIEDB_VERSION"
echo "Build directory: $BUILD_DIR"
echo "Output directory: $OUTPUT_DIR"

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

# Clone or update repository
if [ -d "reth-triedb" ]; then
    echo "Updating existing reth-triedb repository..."
    cd reth-triedb
    git fetch origin
else
    echo "Cloning reth-triedb repository..."
    git clone "$RETH_TRIEDB_REPO" reth-triedb
    cd reth-triedb
fi

# Checkout specified version/commit
echo "Checking out version: $RETH_TRIEDB_VERSION"
git checkout "$RETH_TRIEDB_VERSION"

# Build the binary using cargo
echo "Building reth-triedb with cargo..."
cargo build --release --bin op-reth --manifest-path crates/optimism/bin/Cargo.toml

# Copy binary to output directory
echo "Copying binary to output directory..."
mkdir -p "../../$OUTPUT_DIR"

# Prefer a binary named reth-triedb if the repo provides it; otherwise copy reth and rename
if [ -f target/release/op-reth ]; then
    cp target/release/op-reth "../../$OUTPUT_DIR/reth-triedb"
elif [ -f target/release/reth ]; then
    cp target/release/op-reth "../../$OUTPUT_DIR/reth-triedb"
else
    echo "Error: Built binary not found (expected target/release/op-reth or target/release/reth-triedb)" >&2
    exit 1
fi

echo "reth-triedb binary built successfully and placed in $OUTPUT_DIR/reth-triedb" 

 
