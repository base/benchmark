#!/bin/bash

set -e

# Source versions if available, otherwise use defaults
if [ -f "versions.env" ]; then
    source versions.env
else
    # Default values
    RBUILDER_REPO="${RBUILDER_REPO:-https://github.com/haardikk21/op-rbuilder}"
    RBUILDER_VERSION="${RBUILDER_VERSION:-main}"
    BUILD_DIR="${BUILD_DIR:-./build}"
    OUTPUT_DIR="${OUTPUT_DIR:-../bin}"
fi

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
    git fetch origin
else
    echo "Cloning op-rbuilder repository..."
    git clone "$RBUILDER_REPO" op-rbuilder
    cd op-rbuilder
fi

# Checkout specified version/commit
echo "Checking out version: $RBUILDER_VERSION"
git checkout "$RBUILDER_VERSION"

# Build the binary using cargo
echo "Building op-rbuilder with cargo..."
cargo build --release

# Copy binary to output directory
echo "Copying binary to output directory..."
mkdir -p "../../$OUTPUT_DIR"

# Find the built binary and copy it
if [ -f "target/release/op-rbuilder" ]; then
    cp target/release/op-rbuilder "../../$OUTPUT_DIR/"
elif [ -f "target/release/rbuilder" ]; then
    cp target/release/rbuilder "../../$OUTPUT_DIR/op-rbuilder"
else
    echo "Looking for rbuilder binary..."
    find target/release -name "*rbuilder*" -type f -executable | head -1 | xargs -I {} cp {} "../../$OUTPUT_DIR/op-rbuilder"
fi

echo "op-rbuilder binary built successfully and placed in $OUTPUT_DIR/op-rbuilder" 