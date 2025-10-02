#!/bin/bash

set -e

# setup-base-snapshot.sh - Downloads and extracts Base network snapshots
# 
# Downloads the latest snapshot from Base's official snapshot servers and extracts
# it to the specified destination directory. Supports both mainnet and testnet (sepolia).
#
# Requirements: curl, tar, zstd (for .tar.zst files)
#
# Usage: ./setup-base-snapshot.sh --network <network> --node-type <node-type> --destination <destination> [--skip-if-nonempty]
#
# Networks: mainnet, sepolia (testnet)
# Node types: geth (full snapshots), reth (archive snapshots)
# 
# Examples:
#   ./setup-base-snapshot.sh --network mainnet --node-type geth --destination ./geth-data
#   ./setup-base-snapshot.sh --network sepolia --node-type reth --destination ./reth-data --skip-if-nonempty

POSITIONAL_ARGS=()
for arg in "$@"; do
    case $arg in
        --skip-if-nonempty)
            SKIP_IF_NONEMPTY=true
            shift # Remove --skip-if-nonempty from processing
            ;;
        --network)
            NETWORK=$2
            shift 2
            ;;
        --node-type)
            NODE_TYPE=$2
            shift 2
            ;;
        --destination)
            DESTINATION=$2
            shift 2
            ;;
        *)
            POSITIONAL_ARGS+=("$arg") # Save positional argument
            ;;
    esac
done
set -- "${POSITIONAL_ARGS[@]}" # Restore positional parameters
# Check if the correct number of arguments is provided

if [ -z "$NETWORK" ] || [ -z "$NODE_TYPE" ] || [ -z "$DESTINATION" ]; then
    echo "Error: Missing required parameters"
    echo ""
    echo "Usage: $0 --network <network> --node-type <node-type> --destination <destination> [--skip-if-nonempty]"
    echo ""
    echo "Required parameters:"
    echo "  --network <network>        Network to download snapshot for (mainnet, sepolia)"
    echo "  --node-type <node-type>    Node type (geth, reth)"
    echo "  --destination <destination> Directory to extract snapshot to"
    echo ""
    echo "Optional parameters:"
    echo "  --skip-if-nonempty        Skip download if destination already contains data"
    echo ""
    echo "Examples:"
    echo "  $0 --network mainnet --node-type geth --destination ./geth-data"
    echo "  $0 --network sepolia --node-type reth --destination ./reth-data --skip-if-nonempty"
    exit 1
fi


case $NODE_TYPE in
reth)
    echo "Downloading reth archive snapshot for $NETWORK to $DESTINATION"

    # Check if destination already has data
    if [[ -f "$DESTINATION/db/mdbx.dat" ]] && [[ "$SKIP_IF_NONEMPTY" == "true" ]]; then
        echo "Destination is not empty, skipping download."
        exit 0
    fi

    # Determine snapshot URL based on network
    case $NETWORK in
        mainnet)
            SNAPSHOT_URL_BASE="https://mainnet-reth-archive-snapshots.base.org"
            ;;
        sepolia|sepolia-alpha|testnet)
            SNAPSHOT_URL_BASE="https://sepolia-reth-archive-snapshots.base.org"
            ;;
        *)
            echo "Unsupported network for reth: $NETWORK"
            exit 1
            ;;
    esac

    echo "Getting latest snapshot filename..."
    LATEST_SNAPSHOT=$(curl -s "$SNAPSHOT_URL_BASE/latest")
    if [[ -z "$LATEST_SNAPSHOT" ]]; then
        echo "Failed to get latest snapshot filename"
        exit 1
    fi

    echo "Latest snapshot: $LATEST_SNAPSHOT"
    SNAPSHOT_URL="$SNAPSHOT_URL_BASE/$LATEST_SNAPSHOT"

    # Create destination directory
    mkdir -p "$DESTINATION"
    
    echo "Downloading and extracting snapshot..."
    
    if [[ "$LATEST_SNAPSHOT" == *.tar.zst ]]; then
        curl -L --progress-bar "$SNAPSHOT_URL" | zstd -d | tar -xf - -C "$DESTINATION" --strip-components=1
    else
        curl -L --progress-bar "$SNAPSHOT_URL" | tar -xzf - -C "$DESTINATION" --strip-components=1
    fi
    
    if [[ $? -eq 0 ]]; then
        echo "Successfully downloaded and extracted reth snapshot to $DESTINATION"
    else
        echo "Failed to download or extract snapshot"
        exit 1
    fi
    ;;
geth)
    echo "Downloading geth full snapshot for $NETWORK to $DESTINATION"

    # Check if destination already has data
    if [[ -d "$DESTINATION/geth/chaindata" ]] && [[ "$SKIP_IF_NONEMPTY" == "true" ]]; then
        echo "Destination is not empty, skipping download."
        exit 0
    fi

    # Determine snapshot URL based on network
    case $NETWORK in
        mainnet)
            SNAPSHOT_URL_BASE="https://mainnet-full-snapshots.base.org"
            ;;
        sepolia|sepolia-alpha|testnet)
            SNAPSHOT_URL_BASE="https://sepolia-full-snapshots.base.org"
            ;;
        *)
            echo "Unsupported network for geth: $NETWORK"
            exit 1
            ;;
    esac

    echo "Getting latest snapshot filename..."
    LATEST_SNAPSHOT=$(curl -s "$SNAPSHOT_URL_BASE/latest")
    if [[ -z "$LATEST_SNAPSHOT" ]]; then
        echo "Failed to get latest snapshot filename"
        exit 1
    fi

    echo "Latest snapshot: $LATEST_SNAPSHOT"
    SNAPSHOT_URL="$SNAPSHOT_URL_BASE/$LATEST_SNAPSHOT"

    # Create destination directory
    mkdir -p "$DESTINATION"
    
    # Download and extract snapshot directly to destination
    echo "Downloading and extracting snapshot..."
    
    if [[ "$LATEST_SNAPSHOT" == *.tar.zst ]]; then
        curl -L --progress-bar "$SNAPSHOT_URL" | zstd -d | tar -xf - -C "$DESTINATION" --strip-components=1
    else
        curl -L --progress-bar "$SNAPSHOT_URL" | tar -xzf - -C "$DESTINATION" --strip-components=1
    fi
    
    if [[ $? -eq 0 ]]; then
        echo "Successfully downloaded and extracted geth snapshot to $DESTINATION"
    else
        echo "Failed to download or extract snapshot"
        exit 1
    fi
    ;;
*)
    echo "Unknown node type: $NODE_TYPE"
    echo "Supported node types: geth, reth"
    exit 1
    ;;
esac
