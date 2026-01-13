#!/bin/bash

set -e

# Source versions if available, otherwise use defaults
if [ -f "versions.env" ]; then
    source versions.env
fi

# Default values
RBUILDER_REPO="${RBUILDER_REPO:-https://github.com/base/op-rbuilder}"
RBUILDER_VERSION="${RBUILDER_VERSION:-main}"
BUILD_DIR="${BUILD_DIR:-./build}"
OUTPUT_DIR="${OUTPUT_DIR:-../bin}"

echo "Building op-rbuilder binary..."
echo "Repository: $RBUILDER_REPO"
echo "Version/Commit: $RBUILDER_VERSION"
echo "Build directory: $BUILD_DIR"
echo "Output directory: $OUTPUT_DIR"

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

# Clone or update repository
if [ -d "op-rbuilder" ]; then
    echo "Updating existing op-rbuilder repository..."
    cd op-rbuilder

    # ensure remote matches the repository
    git remote set-url origin "$RBUILDER_REPO"
    git fetch origin
else
    echo "Cloning op-rbuilder repository..."
    git clone "$RBUILDER_REPO" op-rbuilder
    cd op-rbuilder
fi

# Checkout specified version/commit
echo "Checking out version: $RBUILDER_VERSION"
git checkout -f "$RBUILDER_VERSION"

# Build the binary using cargo
echo "Building op-rbuilder with cargo..."
cargo build -p op-rbuilder --bin op-rbuilder --release

# Copy binary to output directory
echo "Copying binary to output directory..."
# Handle absolute paths correctly
if [[ "$OUTPUT_DIR" == /* ]]; then
    # Absolute path - use directly
    FINAL_OUTPUT_DIR="$OUTPUT_DIR"
else
    # Relative path - resolve from current location (clients/build/op-rbuilder)
    FINAL_OUTPUT_DIR="../../$OUTPUT_DIR"
fi
mkdir -p "$FINAL_OUTPUT_DIR"

# Find the built binary and copy it
if [ -f "target/release/op-rbuilder" ]; then
    cp target/release/op-rbuilder "$FINAL_OUTPUT_DIR/"
elif [ -f "target/release/rbuilder" ]; then
    cp target/release/rbuilder "$FINAL_OUTPUT_DIR/op-rbuilder"
else
    echo "No op-rbuilder binary found"
    exit 1
fi

echo "op-rbuilder binary built successfully and placed in $FINAL_OUTPUT_DIR/op-rbuilder" 