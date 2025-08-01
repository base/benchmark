#!/bin/bash

set -e

# setup-snapshot.sh [--skip-if-nonempty] <node-type> <destination>
# Copies a snapshot to the destination if it does not exist.

# Usage: ./setup-snapshot.sh <node-type> <destination>

POSITIONAL_ARGS=()
for arg in "$@"; do
    case $arg in
        --skip-if-nonempty)
            SKIP_IF_NONEMPTY=true
            shift # Remove --skip-if-nonempty from processing
            ;;
        *)
            POSITIONAL_ARGS+=("$arg") # Save positional argument
            ;;
    esac
done
set -- "${POSITIONAL_ARGS[@]}" # Restore positional parameters
# Check if the correct number of arguments is provided

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 [--skip-if-nonempty] <node-type> <destination>"
    exit 1
fi

NODE_TYPE=$1
DESTINATION=$2

# TODO: inject with env var
RETH_SNAPSHOT_LOCATION="/data/snapshots"

case $NODE_TYPE in
reth)
    echo "Copying reth snapshot to $DESTINATION"

    if [[ -f "$DESTINATION/db/mdbx.dat" ]]; then
        echo "Destination is not empty, skipping copy."
        exit 0
    else
        echo "Does not exist, copying snapshot."
        mkdir -p "$DESTINATION"
        rsync -q -r "$RETH_SNAPSHOT_LOCATION/" "$DESTINATION/"
    fi
    ;;
*)
    echo "Unknown node type: $NODE_TYPE"
    exit 1
    ;;
esac
