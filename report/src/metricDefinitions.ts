import { ChartConfig } from "./types"; // Import from types.ts
export const CHART_CONFIG = {
  "latency/send_txs": {
    type: "line",
    group: "Latency",
    title: "Send Txs",
    description: "Shows the median time taken for send txs",
    unit: "ns",
  },
  "latency/update_fork_choice": {
    type: "line",
    group: "Latency",
    title: "Update Fork Choice",
    description: "Shows the median time taken for update fork choice",
    unit: "ns",
  },
  "latency/get_payload": {
    type: "line",
    group: "Latency",
    title: "Get Payload",
    description: "Shows the median time taken for get payload",
    unit: "ns",
  },
  "latency/new_payload": {
    type: "line",
    group: "Latency",
    title: "New Payload",
    description: "Shows the median time taken for new payload",
    unit: "ns",
  },
  "chain/inserts.50-percentile": {
    type: "line",
    group: "Chain",
    title: "Inserts",
    description:
      "Shows the median time taken for block processing and insertion (end-to-end)",
    unit: "ns",
  },
  "chain/account/reads.50-percentile": {
    // Added
    type: "line",
    title: "Account Reads",
    description:
      "Shows the median time taken for account reads during block processing",
    unit: "ns",
  },
  "chain/storage/reads.50-percentile": {
    type: "line",
    group: "Chain",
    title: "Storage Reads",
    description:
      "Shows the median time taken for storage reads during block processing",
    unit: "ns",
  },
  "chain/execution.50-percentile": {
    // Added
    type: "line",
    title: "Execution (EVM)",
    description:
      "Shows the median time taken for EVM execution during block processing",
    unit: "ns",
  },
  "chain/account/updates.50-percentile": {
    type: "line",
    group: "Chain",
    title: "Account Updates",
    description:
      "Shows the median time taken for updating accounts during state validation",
    unit: "ns",
  },
  "chain/account/hashes.50-percentile": {
    type: "line",
    group: "Chain",
    title: "Account Hashes",
    description:
      "Shows the median time taken for hashing accounts during state validation",
    unit: "ns",
  },
  "chain/storage/updates.50-percentile": {
    type: "line",
    group: "Chain",
    title: "Storage Updates", // Renamed from 'Storage Writes' for consistency
    description:
      "Shows the median time taken for updating storage during state validation",
    unit: "ns",
  },
  "chain/validation.50-percentile": {
    type: "line",
    group: "Chain",
    title: "Validation (Misc)",
    description:
      "Shows the median time taken for miscellaneous block validation steps",
    unit: "ns",
  },
  "chain/crossvalidation.50-percentile": {
    // Added
    type: "line",
    title: "Cross Validation",
    description:
      "Shows the median time taken for stateless cross-validation (if enabled)",
    unit: "ns",
  },
  "chain/write.50-percentile": {
    type: "line",
    group: "Chain",
    title: "Write (Misc)",
    description:
      "Shows the median time taken for miscellaneous block write operations (excluding commits)",
    unit: "ns",
  },
  "chain/account/commits.50-percentile": {
    type: "line",
    group: "Chain",
    title: "Account Commits",
    description:
      "Shows the median time taken for committing account changes to the DB",
    unit: "ns",
  },
  "chain/storage/commits.50-percentile": {
    type: "line",
    group: "Chain",
    title: "Storage Commits",
    description:
      "Shows the median time taken for committing storage changes to the DB",
    unit: "ns",
  },
  "chain/snapshot/commits.50-percentile": {
    type: "line",
    group: "Chain",
    title: "Snapshot Commits",
    description:
      "Shows the median time taken for committing snapshot changes to the DB",
    unit: "ns",
  },
  "chain/triedb/commits.50-percentile": {
    type: "line",
    group: "Chain",
    title: "TrieDB Commits",
    description: "Shows the median time taken for committing TrieDB changes",
    unit: "ns",
  },
  "transactions/per_block": {
    type: "line",
    group: "Throughput",
    title: "Transactions per Block",
    description: "Shows the number of transactions per block",
    unit: "count",
  },
  "gas/per_block": {
    type: "line",
    group: "Throughput",
    title: "Gas Per Block",
    description: "Shows the median gas per block",
    unit: "gas",
  },
  reth_sync_execution_execution_duration_avg: {
    type: "line",
    title: "Reth Sync Execution Duration",
    description: "Shows the average time taken for execution during reth sync",
    unit: "s",
    aliases: ["reth_sync_execution_execution_duration"],
  },
  reth_sync_block_validation_state_root_duration_avg: {
    type: "line",
    title: "Reth Sync Block Validation State Root Duration",
    description:
      "Shows the average time taken for state root validation during reth sync",
    unit: "s",
    aliases: ["reth_sync_block_validation_state_root_duration"],
  },
  reth_base_builder_block_built_success: {
    type: "line",
    title: "Builder Block Built Success",
    description: "Number of blocks successfully built per block interval",
    unit: "count",
    aliases: ["reth_op_rbuilder_block_built_success"],
  },
  reth_base_builder_flashblock_count: {
    type: "line",
    title: "Builder Flashblock Count",
    description: "Number of flashblock bundles sent per block",
    unit: "count",
    aliases: ["reth_op_rbuilder_flashblock_count"],
  },
  reth_base_builder_flashblock_count_avg: {
    type: "line",
    title: "Builder Flashblocks per Block (avg)",
    description: "Average number of flashblocks included per block",
    unit: "count",
  },
  reth_base_builder_total_block_built_duration_avg: {
    type: "line",
    title: "Builder Total Block Built Duration",
    description: "Average total time taken to build a block",
    unit: "s",
    aliases: ["reth_op_rbuilder_total_block_built_duration"],
  },
  reth_base_builder_flashblock_build_duration_avg: {
    type: "line",
    title: "Builder Flashblock Build Duration",
    description: "Average time taken to build a single flashblock",
    unit: "s",
    aliases: ["reth_op_rbuilder_flashblock_build_duration"],
  },
  reth_base_builder_state_root_calculation_duration_avg: {
    type: "line",
    title: "Builder State Root Calculation Duration",
    description: "Average time taken to calculate the state root",
    unit: "s",
    aliases: ["reth_op_rbuilder_state_root_calculation_duration"],
  },
  reth_base_builder_state_root_calculation_duration_quantile_0_5: {
    type: "line",
    title: "Builder State Root Calculation Duration p50",
    description: "p50 time taken to calculate the state root",
    unit: "s",
  },
  reth_base_builder_state_root_calculation_duration_quantile_0_9: {
    type: "line",
    title: "Builder State Root Calculation Duration p90",
    description: "p90 time taken to calculate the state root",
    unit: "s",
  },
  reth_base_builder_state_root_calculation_duration_quantile_0_99: {
    type: "line",
    title: "Builder State Root Calculation Duration p99",
    description: "p99 time taken to calculate the state root",
    unit: "s",
  },
  reth_base_builder_state_root_time_per_gas_ratio_quantile_0_9: {
    type: "line",
    title: "Builder State Root Time per Gas p90",
    description: "p90 state-root calculation time divided by gas processed",
  },
  reth_base_builder_sequencer_tx_duration_avg: {
    type: "line",
    title: "Builder Sequencer Tx Duration",
    description: "Average time taken to process sequencer transactions",
    unit: "s",
    aliases: ["reth_op_rbuilder_sequencer_tx_duration"],
  },
  reth_base_builder_payload_transaction_simulation_duration_avg: {
    type: "line",
    title: "Builder Payload Tx Simulation Duration",
    description: "Average time taken for payload transaction simulation",
    unit: "s",
    aliases: ["reth_op_rbuilder_payload_tx_simulation_duration"],
  },
  reth_base_builder_tx_simulation_duration_avg: {
    type: "line",
    title: "Builder Tx Simulation Duration",
    description: "Average per-transaction simulation duration",
    unit: "s",
  },
  reth_base_builder_payload_num_tx_gauge: {
    type: "line",
    title: "Builder Payload Tx Count",
    description: "Number of transactions included in the most recent payload",
    unit: "count",
  },
  reth_base_builder_flashblock_gas_headroom_pct_avg: {
    type: "line",
    title: "Builder Flashblock Gas Headroom %",
    description: "Average gas headroom percentage across flashblocks",
    unit: "count",
  },
  reth_storage_providers_database_save_blocks_total_quantile_0_9: {
    type: "line",
    title: "Save Blocks Total p90",
    description: "p90 total database save-blocks duration",
    unit: "s",
  },
  reth_storage_providers_database_save_blocks_block_count_last: {
    type: "line",
    title: "Save Blocks Block Count",
    description:
      "Number of blocks included in the most recent save-blocks operation",
    unit: "blocks",
  },
  reth_storage_providers_database_save_blocks_commit_sf_quantile_0_9: {
    type: "line",
    title: "Save Blocks Static File Commit p90",
    description: "p90 static-file commit duration during save-blocks",
    unit: "s",
  },
  reth_storage_providers_database_save_blocks_commit_mdbx_quantile_0_9: {
    type: "line",
    title: "Save Blocks MDBX Commit p90",
    description: "p90 MDBX commit duration during save-blocks",
    unit: "s",
  },
  reth_storage_providers_database_save_blocks_write_state_quantile_0_9: {
    type: "line",
    title: "Save Blocks Write State p90",
    description: "p90 state write duration during save-blocks",
    unit: "s",
  },
  reth_storage_providers_database_save_blocks_write_hashed_state_quantile_0_9: {
    type: "line",
    title: "Save Blocks Write Hashed State p90",
    description: "p90 hashed-state write duration during save-blocks",
    unit: "s",
  },
  reth_storage_providers_database_save_blocks_write_trie_updates_quantile_0_9: {
    type: "line",
    title: "Save Blocks Write Trie Updates p90",
    description: "p90 trie-update write duration during save-blocks",
    unit: "s",
  },
  reth_storage_providers_database_save_blocks_sf_quantile_0_9: {
    type: "line",
    title: "Save Blocks Static Files p90",
    description: "p90 static-file save-blocks duration",
    unit: "s",
  },
  reth_trie_leaves_added_quantile_0_9: {
    type: "line",
    title: "Trie Leaves Added p90",
    description: "p90 trie leaves added",
    unit: "count",
  },
  reth_trie_branches_added_quantile_0_9: {
    type: "line",
    title: "Trie Branches Added p90",
    description: "p90 trie branches added",
    unit: "count",
  },
  reth_tree_root_sparse_trie_total_duration_histogram_quantile_0_9: {
    type: "line",
    title: "Sparse Trie Total Duration p90",
    description: "p90 sparse-trie total duration",
    unit: "s",
    aliases: ["reth_tree_root_sparse_trie_total_duration_histogram"],
  },
  reth_tree_root_sparse_trie_final_update_duration_histogram_quantile_0_9: {
    type: "line",
    title: "Sparse Trie Final Update Duration p90",
    description: "p90 sparse-trie final update duration",
    unit: "s",
    aliases: ["reth_tree_root_sparse_trie_final_update_duration_histogram"],
  },
  reth_parallel_sparse_trie_subtrie_hash_update_latency_quantile_0_9: {
    type: "line",
    title: "Sparse Trie Subtrie Hash Update p90",
    description: "p90 subtrie hash update latency",
    unit: "s",
  },
  reth_parallel_sparse_trie_subtrie_upper_hash_latency_quantile_0_9: {
    type: "line",
    title: "Sparse Trie Subtrie Upper Hash p90",
    description: "p90 subtrie upper-hash latency",
    unit: "s",
  },
  reth_trie_proof_task_storage_worker_idle_time_seconds_quantile_0_9: {
    type: "line",
    title: "Trie Proof Storage Worker Idle p90",
    description: "p90 trie-proof storage worker idle time",
    unit: "s",
  },
  reth_trie_proof_task_account_worker_idle_time_seconds_quantile_0_9: {
    type: "line",
    title: "Trie Proof Account Worker Idle p90",
    description: "p90 trie-proof account worker idle time",
    unit: "s",
  },
  reth_trie_proof_task_blinded_storage_nodes_quantile_0_9: {
    type: "line",
    title: "Trie Proof Blinded Storage Nodes p90",
    description: "p90 blinded storage nodes handled by trie proof tasks",
    unit: "count",
  },
  reth_trie_proof_task_blinded_account_nodes_quantile_0_9: {
    type: "line",
    title: "Trie Proof Blinded Account Nodes p90",
    description: "p90 blinded account nodes handled by trie proof tasks",
    unit: "count",
  },
  reth_trie_cursor_overall_duration_quantile_0_9: {
    type: "line",
    title: "Trie Cursor Overall Duration p90",
    description: "p90 trie cursor overall duration",
    unit: "s",
  },
  reth_trie_hashed_cursor_overall_duration_quantile_0_9: {
    type: "line",
    title: "Trie Hashed Cursor Overall Duration p90",
    description: "p90 hashed trie cursor overall duration",
    unit: "s",
  },
  reth_db_freelist: {
    type: "line",
    title: "MDBX Freelist",
    description: "MDBX freelist size",
    unit: "count",
  },
  reth_sync_state_provider_total_storage_fetch_latency_avg: {
    type: "line",
    title: "Validator Storage Load Latency",
    description: "Average latency for storage slot loads during validation",
    unit: "s",
    aliases: ["reth_sync_state_provider_total_storage_fetch_latency"],
  },
  reth_sync_state_provider_total_code_fetch_latency_avg: {
    type: "line",
    title: "Validator Code Load Latency",
    description: "Average latency for bytecode loads during validation",
    unit: "s",
    aliases: ["reth_sync_state_provider_total_code_fetch_latency"],
  },
  reth_sync_state_provider_total_account_fetch_latency_avg: {
    type: "line",
    title: "Validator Account Load Latency",
    description: "Average latency for account loads during validation",
    unit: "s",
    aliases: ["reth_sync_state_provider_total_account_fetch_latency"],
  },
} satisfies Record<string, ChartConfig>;

const CHART_CONFIG_ORDER: (keyof typeof CHART_CONFIG)[] = [
  "latency/get_payload",
  "latency/new_payload",
  "latency/update_fork_choice",
  "latency/send_txs",
  "gas/per_block",
  "transactions/per_block",
  "chain/inserts.50-percentile",
  "chain/account/reads.50-percentile",
  "chain/storage/reads.50-percentile",
  "chain/execution.50-percentile",
  "chain/account/updates.50-percentile",
  "chain/account/hashes.50-percentile",
  "chain/storage/updates.50-percentile",
  "chain/validation.50-percentile",
  "chain/crossvalidation.50-percentile",
  "chain/write.50-percentile",
  "chain/account/commits.50-percentile",
  "chain/storage/commits.50-percentile",
  "chain/snapshot/commits.50-percentile",
  "chain/triedb/commits.50-percentile",
  "reth_base_builder_state_root_calculation_duration_quantile_0_5",
  "reth_base_builder_state_root_calculation_duration_quantile_0_9",
  "reth_base_builder_state_root_calculation_duration_quantile_0_99",
  "reth_base_builder_state_root_time_per_gas_ratio_quantile_0_9",
  "reth_storage_providers_database_save_blocks_total_quantile_0_9",
  "reth_storage_providers_database_save_blocks_block_count_last",
  "reth_storage_providers_database_save_blocks_commit_sf_quantile_0_9",
  "reth_storage_providers_database_save_blocks_commit_mdbx_quantile_0_9",
  "reth_storage_providers_database_save_blocks_write_state_quantile_0_9",
  "reth_storage_providers_database_save_blocks_write_hashed_state_quantile_0_9",
  "reth_storage_providers_database_save_blocks_write_trie_updates_quantile_0_9",
  "reth_storage_providers_database_save_blocks_sf_quantile_0_9",
  "reth_trie_leaves_added_quantile_0_9",
  "reth_trie_branches_added_quantile_0_9",
  "reth_tree_root_sparse_trie_total_duration_histogram_quantile_0_9",
  "reth_tree_root_sparse_trie_final_update_duration_histogram_quantile_0_9",
  "reth_parallel_sparse_trie_subtrie_hash_update_latency_quantile_0_9",
  "reth_parallel_sparse_trie_subtrie_upper_hash_latency_quantile_0_9",
  "reth_trie_proof_task_storage_worker_idle_time_seconds_quantile_0_9",
  "reth_trie_proof_task_account_worker_idle_time_seconds_quantile_0_9",
  "reth_trie_proof_task_blinded_storage_nodes_quantile_0_9",
  "reth_trie_proof_task_blinded_account_nodes_quantile_0_9",
  "reth_trie_cursor_overall_duration_quantile_0_9",
  "reth_trie_hashed_cursor_overall_duration_quantile_0_9",
  "reth_db_freelist",
];

export const SORTED_CHART_CONFIG: [string, ChartConfig][] = Object.entries(
  CHART_CONFIG,
).sort((a, b) => {
  const aIndex = CHART_CONFIG_ORDER.indexOf(a[0] as keyof typeof CHART_CONFIG);
  const bIndex = CHART_CONFIG_ORDER.indexOf(b[0] as keyof typeof CHART_CONFIG);

  // if both doesn't exist, sort it last (infinity)
  if (aIndex === -1 && bIndex === -1) {
    return 0;
  }
  if (aIndex === -1) {
    return 1;
  }
  if (bIndex === -1) {
    return -1;
  }
  return aIndex - bIndex;
});
