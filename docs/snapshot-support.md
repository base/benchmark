# Snapshot Support

## Problem

We want to support running benchmarks from a snapshot.

## Proposed Schema

```yaml
- name: Transfer-only execution speed
  description: Transfer-only execution speed
  snapshot:
    command: setup_snapshot.sh # setup_snapshot.sh <node_type> <data_dir>
  variables:
    - type: transaction_workload
      values:
        - transfer-only
    - type: node_type
      values:
        - geth
        - reth
    - type: num_blocks
      value: 10
    - type: gas_limit
      values:
        - 15000000
        - 30000000
        - 60000000
        - 90000000
```

```bash
#!/bin/bash

# setup-snapshot <node_type> <data_dir>

case $1 in
  geth)
    # download/copy over geth snapshot
    ;;
  reth)
    # download/copy over reth snapshot
    ;;
  *)
    echo "Invalid node type"
    exit 1
    ;;
esac
```
