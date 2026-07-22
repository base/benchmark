#!/usr/bin/env bash
# zfs-clone-mainnet.sh <node_type> <snapshot_path>
#
# Clones the base-mainnet-06-09 ZFS snapshot to <snapshot_path> for use as a
# benchmark data directory. Called by base-bench's snapshot manager before
# each run. Idempotent: destroys any previous clone at the same path first.
#
# Prerequisites:
#   - ZFS snapshot zroot/data/snapshots/base-mainnet-b20-bench@1m-populated must exist
#   - Must be run as a user with ZFS clone/destroy privileges (e.g. root or
#     delegated via `zfs allow`)
#
# Usage (invoked automatically by base-bench):
#   zfs-clone-mainnet.sh builder /path/to/snapshots/builder_sequencer_abc123def456

set -euo pipefail

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <node_type> <snapshot_path>" >&2
    exit 1
fi

NODE_TYPE="$1"
SNAPSHOT_PATH="$2"

SOURCE_SNAPSHOT="zroot/data/snapshots/base-mainnet-b20-bench@1m-populated"
BASENAME=$(basename "$SNAPSHOT_PATH")
DATASET="zroot/data/snapshots/bench-${BASENAME}"

# Destroy any existing dataset mounted at snapshot_path
EXISTING=$(zfs list -H -o name,mountpoint 2>/dev/null \
    | awk -v mp="$SNAPSHOT_PATH" '$2 == mp { print $1 }')
if [ -n "$EXISTING" ]; then
    echo "Destroying existing ZFS dataset $EXISTING (mounted at $SNAPSHOT_PATH)"
    sudo zfs destroy "$EXISTING"
fi

# Also destroy by derived name in case it's mounted elsewhere
if zfs list "$DATASET" &>/dev/null; then
    echo "Destroying stale ZFS dataset $DATASET"
    sudo zfs destroy "$DATASET"
fi

rmdir "$SNAPSHOT_PATH" 2>/dev/null || true

echo "Cloning $SOURCE_SNAPSHOT → $DATASET (mountpoint: $SNAPSHOT_PATH)"
sudo zfs clone -o mountpoint="$SNAPSHOT_PATH" "$SOURCE_SNAPSHOT" "$DATASET"

echo "ZFS clone ready at $SNAPSHOT_PATH"
