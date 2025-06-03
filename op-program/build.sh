#!/bin/bash

# In order to benchmark op-program, we need to build op-program with our own chain ID.
# This script will clone the OP repo, install the necessary configuration files,
# and build the op-program binary.

CHAIN_ID=13371337
OP_PROGRAM_VERSION="v1.6.1-rc.1"

# ensure running in op-program directory
if [ ! -d "../op-program" ]; then
    echo "Please run this script from the op-program directory."
    exit 1
fi

# clone OP repo
if [ ! -d "op" ]; then
    git clone https://github.com/ethereum-optimism/optimism.git
    git -C optimism checkout op-program/$OP_PROGRAM_VERSION
    echo "Cloned OP repo at version op-program/$OP_PROGRAM_VERSION."
else
    echo "OP repo already exists, skipping clone."
fi

pushd optimism

# install rollup.json and genesis.json
echo "Installing rollup.json and genesis.json for chain ID $CHAIN_ID..."
cp ../../rollup.json op-program/chainconfig/configs/13371337-rollup.json
cp ../../genesis.json op-program/chainconfig/configs/13371337-genesis-l2.json

# update git submodules
echo "Updating git submodules..."
git submodule update --init --recursive

# build contracts
echo "Building contracts..."
pushd packages/contracts-bedrock
forge build
popd

# build op-program
echo "Building op-program..."
make op-program

# copy op-program binary to versions directory
mkdir -p ../versions/$OP_PROGRAM_VERSION
cp op-program/bin/op-program ../versions/$OP_PROGRAM_VERSION
popd