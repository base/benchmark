#!/bin/bash

set -e

# Source versions if available, otherwise use defaults
if [ -f "versions.env" ]; then
    source versions.env
fi

# Default values
OPTIMISM_REPO="${OPTIMISM_REPO:-https://github.com/ethereum-optimism/optimism/}"
OPTIMISM_VERSION="${OPTIMISM_VERSION:-develop}"
BUILD_DIR="${BUILD_DIR:-./build}"
OUTPUT_DIR="${OUTPUT_DIR:-../bin}"

echo "Building op-reth binary..."
echo "Repository: $OPTIMISM_REPO"
echo "Version/Commit: $OPTIMISM_VERSION"
echo "Build directory: $BUILD_DIR"
echo "Output directory: $OUTPUT_DIR"

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

# Clone or update repository
if [ -d "optimism" ]; then
    echo "Updating existing optimism repository..."
    cd optimism
    git fetch origin

    # ensure remote matches the repository
    git remote set-url origin "$OPTIMISM_REPO"
    git fetch origin
else
    echo "Cloning optimism repository..."
    git clone "$OPTIMISM_REPO" optimism
    cd optimism
fi

# Checkout specified version/commit
echo "Checking out version: $OPTIMISM_VERSION"
git checkout -f "$OPTIMISM_VERSION"

pushd rust

# Build the binary using cargo
echo "Building op-reth with cargo..."
# Build with performance features matching CI workflow
cargo build --features asm-keccak,jemalloc --bin op-reth --profile maxperf --manifest-path op-reth/bin/Cargo.toml

popd

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
cp rust/target/maxperf/op-reth "$FINAL_OUTPUT_DIR/"

echo "op-reth binary built successfully and placed in $FINAL_OUTPUT_DIR/op-reth" 
