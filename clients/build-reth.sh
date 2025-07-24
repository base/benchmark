#!/bin/bash

set -e

# Source versions if available, otherwise use defaults
if [ -f "versions.env" ]; then
    source versions.env
else
    # Default values
    RETH_REPO="${RETH_REPO:-https://github.com/paradigmxyz/reth/}"
    RETH_VERSION="${RETH_VERSION:-main}"
    BUILD_DIR="${BUILD_DIR:-./build}"
    OUTPUT_DIR="${OUTPUT_DIR:-../bin}"
fi

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
else
    echo "Cloning reth repository..."
    git clone "$RETH_REPO" reth
    cd reth
fi

# Checkout specified version/commit
echo "Checking out version: $RETH_VERSION"
git checkout "$RETH_VERSION"

# Build the binary using cargo
echo "Building reth with cargo..."
cargo build --release --bin reth

# Copy binary to output directory
echo "Copying binary to output directory..."
mkdir -p "../../$OUTPUT_DIR"
cp target/release/reth "../../$OUTPUT_DIR/"

echo "reth binary built successfully and placed in $OUTPUT_DIR/reth" 
