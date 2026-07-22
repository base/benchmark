#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <node_type> <snapshot_path>" >&2
    exit 1
fi

NODE_TYPE="$1"
SNAPSHOT_PATH="$2"

SOURCE_SNAPSHOT="zroot/data/snapshots/base-mainnet-b20-10m@10m-populated"
BASENAME=$(basename "$SNAPSHOT_PATH")
DATASET="zroot/data/snapshots/bench-${BASENAME}"

EXISTING=$(zfs list -H -o name,mountpoint 2>/dev/null \
    | awk -v mp="$SNAPSHOT_PATH" '$2 == mp { print $1 }')
if [ -n "$EXISTING" ]; then
    echo "Destroying existing ZFS dataset $EXISTING (mounted at $SNAPSHOT_PATH)"
    sudo zfs destroy "$EXISTING"
fi

if zfs list "$DATASET" &>/dev/null; then
    echo "Destroying stale ZFS dataset $DATASET"
    sudo zfs destroy "$DATASET"
fi

rmdir "$SNAPSHOT_PATH" 2>/dev/null || true

echo "Cloning $SOURCE_SNAPSHOT -> $DATASET (mountpoint: $SNAPSHOT_PATH)"
sudo zfs clone -o mountpoint="$SNAPSHOT_PATH" "$SOURCE_SNAPSHOT" "$DATASET"

echo "ZFS clone ready at $SNAPSHOT_PATH"
