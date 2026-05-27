# Separating sequencer and validator benchmark

- Allow block builder and validator to be different node types (geth/reth)

## Block building benchmark

- Processing loop
    - Measure: start time
    - ForkChoiceUpdated w/ NoTxPool: false
    - If internal benchmark: Send transactions to mempool
    - If external benchmark: Transactions are sent to RPC endpoint
    - GetPayload
    - Measure: end time
    - Collect block metrics
    
## Syncing/validating benchmark

- Processing loop
    - Measure: start time
    - NewPayload with generated payloads from sequencer benchmark
    - GetPayload
    - Measure: end time
    - Collect block metrics
- Reason we don't need to test mempool for validating node: only used for tx gossip, no logic actually has to be executed

## Role selection

Benchmark definitions always run the sequencer role. The sequencer phase builds the payloads used by the rest of the benchmark.

By default, benchmarks also run the validator role after the sequencer phase:

```yaml
benchmarks:
  - variables:
      # ...
```

Set `roles: [sequencer]` when a benchmark only needs block-building or snapshot startup coverage and does not need to validate the generated payloads:

```yaml
benchmarks:
  - roles: [sequencer]
    variables:
      # ...
```

The validator role cannot run by itself because validator benchmarks consume payloads produced by the sequencer phase. Proof-program benchmarks also require the validator role.

## op-challenger test

- batch all blocks in the test to L1
- run op-program on those batches - verify output root
