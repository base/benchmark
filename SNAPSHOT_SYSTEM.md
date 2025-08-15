# Two-Tier Snapshot System

## Overview

The benchmarking framework now supports an optimized two-tier snapshot system that significantly improves performance and reduces network overhead when running multiple tests with snapshots.

## Architecture

### Tier 1: Initial Snapshots
- **Purpose**: Downloaded once at benchmark startup and stored persistently
- **Location**: Typically `/data/snapshots/initial_<nodeType>_<hash>`
- **Lifecycle**: Created at benchmark startup, persisted across all tests
- **Usage**: Serves as the source for per-test copies

### Tier 2: Per-Test Snapshots  
- **Purpose**: Test-specific copies created from initial snapshots
- **Location**: Test-specific temporary directories
- **Lifecycle**: Created before each test, cleaned up after test completion
- **Usage**: Isolated environment for each test run

## Configuration Format

### New YAML Structure

```yaml
benchmarks:
  - initial_snapshots:
      - node_type: reth
        command: ./scripts/setup-initial-snapshot.sh --network=sepolia --node-type=reth
        superchain_chain_id: 84532
      - node_type: geth
        command: ./scripts/setup-initial-snapshot.sh --network=sepolia --node-type=geth  
        superchain_chain_id: 84532
    variables:
      - type: node_type
        values: [reth, geth]
      # ... other variables
```

## Implementation Details

### Key Components

1. **SnapshotManager Interface** (`benchmark/snapshots.go`)
   - `EnsureInitialSnapshot()`: Creates initial snapshots
   - `GetInitialSnapshotPath()`: Retrieves initial snapshot paths
   - `CopyFromInitialSnapshot()`: Copies using rsync for efficiency

2. **TestDefinition** (`benchmark/definition.go`)
   - `InitialSnapshots []SnapshotDefinition`: Tier 1 snapshots

3. **Service** (`runner/service.go`)
   - `setupInitialSnapshots()`: Runs at benchmark startup
   - `setupInternalDirectories()`: Uses rsync for per-test copies

### Execution Flow

1. **Benchmark Startup**
   ```
   Service.Run() → setupInitialSnapshots() → EnsureInitialSnapshot()
   ```

2. **Per Test Execution**
   ```
   runTest() → setupInternalDirectories() → CopyFromInitialSnapshot()
   ```

3. **Test Cleanup**
   ```
   defer cleanup → os.RemoveAll(testDir) // Removes per-test copies only
   ```

## Usage Examples

### Multi-Node Type Testing
```yaml
initial_snapshots:
  - node_type: reth
    command: ./download-reth-snapshot.sh
  - node_type: geth  
    command: ./download-geth-snapshot.sh
variables:
  - type: node_type
    values: [reth, geth]
```

### Fallback Support
If no initial snapshot exists for a node type, the system automatically falls back to the original single-tier behavior, ensuring backward compatibility.

### Mixed Scenarios
```yaml
# Test plan 1: Uses two-tier system
- initial_snapshots: [...]
  
# Test plan 2: Uses single-tier system  
- initial_snapshots: [...]  # No initial_snapshots
```
