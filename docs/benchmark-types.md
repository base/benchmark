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

## op-challenger test

- batch all blocks in the test to L1
- run op-program on those batches - verify output root