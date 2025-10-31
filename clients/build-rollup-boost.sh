#!/bin/bash

set -e

# Source versions if available, otherwise use defaults
if [ -f "versions.env" ]; then
    source versions.env
else
    # Default values
    ROLLUP_BOOST_REPO="${ROLLUP_BOOST_REPO:-https://github.com/flashbots/rollup-boost.git}"
    ROLLUP_BOOST_VERSION="${ROLLUP_BOOST_VERSION:-main}"
    BUILD_DIR="${BUILD_DIR:-./build}"
    OUTPUT_DIR="${OUTPUT_DIR:-../bin}"
fi

echo "Building rollup-boost binary..."
echo "Repository: $ROLLUP_BOOST_REPO"
echo "Version/Commit: $ROLLUP_BOOST_VERSION"
echo "Build directory: $BUILD_DIR"
echo "Output directory: $OUTPUT_DIR"

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

# Clone or update repository
if [ -d "rollup-boost" ]; then
    echo "Updating existing rollup-boost repository..."
    cd rollup-boost
    git fetch origin
else
    echo "Cloning rollup-boost repository..."
    git clone "$ROLLUP_BOOST_REPO" rollup-boost
    cd rollup-boost
fi

# Checkout specified version/commit
echo "Checking out version: $ROLLUP_BOOST_VERSION"
git checkout "$ROLLUP_BOOST_VERSION"

# Build the binary using cargo
echo "Building rollup-boost with cargo..."
cargo build --bin rollup-boost --release

# Copy binary to output directory
echo "Copying binary to output directory..."
mkdir -p "../../$OUTPUT_DIR"

# Find the built binary and copy it
if [ -f "target/release/rollup-boost" ]; then
    cp target/release/rollup-boost "../../$OUTPUT_DIR/"
    echo "rollup-boost binary built successfully and placed in $OUTPUT_DIR/rollup-boost"
else
    echo "Error: rollup-boost binary not found in target/release/"
    exit 1
fi

echo ""
echo "✓ Rollup-boost built successfully!"
echo ""
echo "To use with dual-builder flashblocks mode:"
echo "  ./bin/base-bench run \\"
echo "    --config ./configs/examples/flashblocks.yml \\"
echo "    --rbuilder-bin ./bin/op-rbuilder \\"
echo "    --flashblocks-fallback reth \\"
echo "    --rollup-boost-bin ./bin/rollup-boost \\"
echo "    --output-dir ./output"

