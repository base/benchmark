#!/bin/bash

set -e

# Source versions if available, otherwise use defaults
if [ -f "versions.env" ]; then
    source versions.env
fi

# Default values
BASE_RETH_NODE_REPO="${BASE_RETH_NODE_REPO:-https://github.com/base/base}"
BASE_RETH_NODE_VERSION="${BASE_RETH_NODE_VERSION:-main}"
BUILD_DIR="${BUILD_DIR:-./build}"
OUTPUT_DIR="${OUTPUT_DIR:-../bin}"

echo "Building base-reth-node and base-builder binaries..."
echo "Repository: $BASE_RETH_NODE_REPO"
echo "Version/Commit: $BASE_RETH_NODE_VERSION"
echo "Build directory: $BUILD_DIR"
echo "Output directory: $OUTPUT_DIR"

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

# Clone or update repository
if [ -d "base" ]; then
    echo "Updating existing base repository..."
    cd base
    git fetch origin

    # ensure remote matches the repository
    git remote set-url origin "$BASE_RETH_NODE_REPO"
    git fetch origin
else
    echo "Cloning base repository..."
    git clone "$BASE_RETH_NODE_REPO" base
    cd base
fi

# Checkout specified version/commit
echo "Checking out version: $BASE_RETH_NODE_VERSION"
git checkout -f "$BASE_RETH_NODE_VERSION"

# Build the binaries using cargo
echo "Building base-reth-node, base-builder, and base-load-test with cargo..."
cargo build --bin base-reth-node --bin base-builder --bin base-load-test --profile maxperf

# Copy binaries to output directory
echo "Copying binaries to output directory..."
# Handle absolute paths correctly
if [[ "$OUTPUT_DIR" == /* ]]; then
    # Absolute path - use directly
    FINAL_OUTPUT_DIR="$OUTPUT_DIR"
else
    # Relative path - resolve from current location (clients/build/base)
    FINAL_OUTPUT_DIR="../../$OUTPUT_DIR"
fi
mkdir -p "$FINAL_OUTPUT_DIR"

# Find the built binaries and copy them
if [ -f "target/maxperf/base-reth-node" ]; then
    cp target/maxperf/base-reth-node "$FINAL_OUTPUT_DIR/"
else
    echo "No base-reth-node binary found"
    exit 1
fi

if [ -f "target/maxperf/base-builder" ]; then
    cp target/maxperf/base-builder "$FINAL_OUTPUT_DIR/"
else
    echo "No base-builder binary found"
    exit 1
fi

if [ -f "target/maxperf/base-load-test" ]; then
    cp target/maxperf/base-load-test "$FINAL_OUTPUT_DIR/"
else
    echo "No base-load-test binary found"
    exit 1
fi

echo "Binaries built successfully and placed in $FINAL_OUTPUT_DIR/"
