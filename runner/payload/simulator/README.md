# Payload Simulator

The Payload Simulator is a tool for analyzing real-world block execution characteristics on Ethereum and L2 chains. It fetches blocks from a live chain via RPC, re-executes them locally using execution witnesses, and outputs detailed statistics about account operations, storage access patterns, opcode usage, and precompile calls.

## Use Cases

- **Benchmark Configuration** - Generate realistic workload parameters based on actual mainnet/testnet data
- **Performance Analysis** - Understand which operations dominate block execution time
- **Precompile Usage** - Identify precompile usage patterns across blocks
- **Capacity Planning** - Analyze storage and account access patterns at scale

## Building

From the repository root:

```bash
go build -o bin/payload-simulator ./runner/payload/simulator/cmd
```

## Usage

> **Note:** The RPC endpoint must support `debug_executionWitness` (for reth) or `debug_dbGet` (for geth) depending on the `--client` flag.

```bash
./bin/payload-simulator \
  --rpc-url <your-rpc-url> \
  --sample-size 100 \
  --num-workers 10 \
  --client reth
```

### Flags

| Flag             | Description                                                                                                                               | Default        |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | -------------- |
| `--rpc-url`      | RPC URL of the chain                                                                                                                      | -              |
| `--sample-size`  | Number of blocks to sample                                                                                                                | `10`           |
| `--sample-range` | Range of blocks to sample from. If equal to `sample-size`, fetches consecutive blocks. If larger, randomly samples blocks from the range. | `sample-size`  |
| `--num-workers`  | Number of parallel workers for fetching and processing                                                                                    | `10`           |
| `--genesis`      | Path to genesis JSON file                                                                                                                 | `genesis.json` |
| `--chain-id`     | Chain ID to load genesis config from (uses OP Stack defaults if set)                                                                      | -              |
| `--client`       | Client type for preimage fetching: `geth` (uses `debug_dbGet`) or `reth` (uses `debug_executionWitness`)                                  | `reth`         |

### Examples

**Analyze the last 100 consecutive blocks using reth:**

```bash
./bin/payload-simulator \
  --rpc-url <your-rpc-url> \
  --sample-size 100 \
  --client reth
```

**Analyze the last 100 consecutive blocks using geth:**

```bash
./bin/payload-simulator \
  --rpc-url <your-rpc-url> \
  --sample-size 100 \
  --client geth
```

**Sample 50 random blocks from the last 10,000 blocks:**

```bash
./bin/payload-simulator \
  --rpc-url <your-rpc-url> \
  --sample-size 50 \
  --sample-range 10000
```

**Use custom genesis file:**

```bash
./bin/payload-simulator \
  --rpc-url <your-rpc-url> \
  --genesis ./custom-genesis.json \
  --sample-size 20
```

## Output Statistics

The simulator outputs aggregate statistics for blocks and transactions:

### Account Operations

- **Accounts Loaded** - Number of account state reads
- **Accounts Deleted** - Number of account deletions
- **Accounts Updated** - Number of account state updates
- **Accounts Created** - Number of new accounts created

### Storage Operations

- **Storage Loaded** - Number of storage slot reads (SLOAD)
- **Storage Deleted** - Number of storage slot deletions
- **Storage Updated** - Number of storage slot writes (SSTORE)
- **Storage Created** - Number of new storage slots created

### Code Metrics

- **Code Size Loaded** - Total bytes of contract code loaded
- **Number of Contracts Loaded** - Count of unique contracts executed

### Opcode Statistics

Tracks usage of expensive opcodes like:

- `EXP` - Exponentiation
- `KECCAK256` - Keccak hashing

### Precompile Statistics

Tracks calls to precompiled contracts:

- `ecrecover` (0x01) - ECDSA recovery
- `sha256hash` (0x02) - SHA-256 hashing
- `ripemd160hash` (0x03) - RIPEMD-160 hashing
- `dataCopy` (0x04) - Identity/data copy
- `bigModExp` (0x05) - Modular exponentiation
- `bn256Add` (0x06) - BN256 curve addition
- `bn256ScalarMul` (0x07) - BN256 scalar multiplication
- `bn256Pairing` (0x08) - BN256 pairing check
- `blake2F` (0x09) - BLAKE2 compression
- `bls12381*` (0x0b-0x11) - BLS12-381 operations
- `p256Verify` (0x0100) - P-256 signature verification

## Example Output

```
Aggregate block stats:
- Accounts Reads: 1523.45
- Accounts Deletes: 0.00
- Accounts Updates: 245.67
- Accounts Created: 12.34
- Storage Reads: 8934.56
- Storage Deletes: 23.45
- Storage Updates: 1234.56
- Storage Created: 456.78
- Code Size Loaded: 234567.89
- Number of Contracts Loaded: 89.12
- Opcode Stats:
   -                  EXP: 123.45
   -             KECCAK256: 5678.90
- Precompile Stats:
   -            ecrecover: 234.56
   -           bn256Add: 12.34

Aggregate tx stats:
- Accounts Reads: 15.23
- Storage Reads: 89.34
...
```

## Requirements

- The RPC endpoint must support the appropriate debug method based on `--client`:
  - **reth**: Requires `debug_executionWitness` method
  - **geth**: Requires `debug_dbGet` method
- Genesis configuration must match the target chain

## How It Works

1. **Block Selection** - Selects block numbers to sample (consecutive or random based on `sample-range`)
2. **Parallel Fetching** - Uses worker pool to fetch blocks and preimage data concurrently
   - **reth mode**: Fetches entire execution witness upfront via `debug_executionWitness`
   - **geth mode**: Fetches preimages on-demand via `debug_dbGet` during execution
3. **Local Re-execution** - Re-executes each block locally using the preimage data
4. **Statistics Collection** - Traces execution to collect account, storage, opcode, and precompile statistics
5. **Aggregation** - Computes per-block and per-transaction averages across all sampled blocks
