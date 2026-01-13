#!/bin/bash

set -e

# Source versions if available, otherwise use defaults
if [ -f "versions.env" ]; then
    source versions.env
fi

# Default values
RETH_REPO="${RETH_REPO:-https://github.com/paradigmxyz/reth/}"
RETH_VERSION="${RETH_VERSION:-main}"
BUILD_DIR="${BUILD_DIR:-./build}"
OUTPUT_DIR="${OUTPUT_DIR:-../bin}"

echo "Building reth binary..."
echo "Repository: $RETH_REPO"
echo "Version/Commit: $RETH_VERSION"
echo "Build directory: $BUILD_DIR"
echo "Output directory: $OUTPUT_DIR"

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

# Clone or update repository
if [ -d "reth" ]; then
    echo "Updating existing reth repository..."
    cd reth
    git fetch origin

    # ensure remote matches the repository
    git remote set-url origin "$RETH_REPO"
    git fetch origin
else
    echo "Cloning reth repository..."
    git clone "$RETH_REPO" reth
    cd reth
fi

# Checkout specified version/commit
echo "Checking out version: $RETH_VERSION"
git checkout -f "$RETH_VERSION"

# Build the binary using cargo
echo "Building reth with cargo..."
# Build with performance features matching CI workflow
cargo build --features asm-keccak,jemalloc --bin op-reth --profile maxperf --manifest-path crates/optimism/bin/Cargo.toml

# Copy binary to output directory
echo "Copying binary to output directory..."
# Handle absolute paths correctly
if [[ "$OUTPUT_DIR" == /* ]]; then
    # Absolute path - use directly
    FINAL_OUTPUT_DIR="$OUTPUT_DIR"
else
    # Relative path - resolve from current location (clients/build/reth)
    FINAL_OUTPUT_DIR="../../$OUTPUT_DIR"
fi
mkdir -p "$FINAL_OUTPUT_DIR"
cp target/maxperf/op-reth "$FINAL_OUTPUT_DIR/"

echo "reth binary built successfully and placed in $FINAL_OUTPUT_DIR/op-reth" 
