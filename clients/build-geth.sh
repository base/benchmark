#!/bin/bash

set -e

# Source versions if available, otherwise use defaults
if [ -f "versions.env" ]; then
    source versions.env
else
    # Default values
    GETH_REPO="${GETH_REPO:-https://github.com/ethereum-optimism/op-geth/}"
    GETH_VERSION="${GETH_VERSION:-optimism}"
    BUILD_DIR="${BUILD_DIR:-./build}"
    OUTPUT_DIR="${OUTPUT_DIR:-../bin}"
fi

echo "Building op-geth binary..."
echo "Repository: $GETH_REPO"
echo "Version/Commit: $GETH_VERSION"
echo "Build directory: $BUILD_DIR"
echo "Output directory: $OUTPUT_DIR"

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

# Clone or update repository
if [ -d "op-geth" ]; then
    echo "Updating existing op-geth repository..."
    cd op-geth
    git fetch origin
else
    echo "Cloning op-geth repository..."
    git clone "$GETH_REPO" op-geth
    cd op-geth
fi

# Checkout specified version/commit
echo "Checking out version: $GETH_VERSION"
git checkout "$GETH_VERSION"

# Build the binary using Go
echo "Building op-geth with Go..."
go run build/ci.go install -static ./cmd/geth

# Copy binary to output directory
echo "Copying binary to output directory..."
mkdir -p "../../$OUTPUT_DIR"

# The binary is typically built in the build directory
if [ -f "build/bin/geth" ]; then
    cp build/bin/geth "../../$OUTPUT_DIR/geth"
elif [ -f "bin/geth" ]; then
    cp bin/geth "../../$OUTPUT_DIR/geth"
else
    echo "Looking for geth binary..."
    find . -name "geth" -type f -executable | head -1 | xargs -I {} cp {} "../../$OUTPUT_DIR/geth"
fi

echo "op-geth binary built successfully and placed in $OUTPUT_DIR/geth" 
