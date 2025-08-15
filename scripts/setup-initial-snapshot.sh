#!/bin/bash

set -e

# setup-initial-snapshot.sh [--skip-if-nonempty] <node-type> <destination>
# Copies a snapshot to the destination if it does not exist.

# Usage: ./setup-initial-snapshot.sh <node-type> <destination>

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

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 [--skip-if-nonempty] [--network <network>] [--node-type <node-type>] [--destination <destination>]"
    exit 1
fi

if [ -z "$NETWORK" ]; then
    echo "Network is required"
    exit 1
fi

if [ -z "$NODE_TYPE" ]; then
    echo "Node type is required"
    exit 1
fi

if [ -z "$DESTINATION" ]; then
    echo "Destination is required"
    exit 1
fi


case $NODE_TYPE in
reth)
    echo "Copying reth snapshot to $DESTINATION"

    mkdir -p "$DESTINATION"
    ./agent_init --gbs-network=$NETWORK --gbs-config-name=base-reth-cbnode --gbs-directory=$DESTINATION
    ;;
geth)
    echo "Copying geth snapshot to $DESTINATION"

    CONFIG_NAME="base-full-cbnode"

    if [[ $NETWORK == "sepolia-alpha" ]]; then
        CONFIG_NAME="base-cbnode"
    fi

    mkdir -p "$DESTINATION"
    ./agent_init --gbs-network=$NETWORK --gbs-config-name=$CONFIG_NAME --gbs-directory=$DESTINATION
    ;;
*)
    echo "Unknown node type: $NODE_TYPE"
    exit 1
    ;;
esac
