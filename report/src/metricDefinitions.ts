import { ChartConfig } from "./types"; // Import from types.ts

export const CHART_CONFIG: Record<string, ChartConfig> = {
  "latency/send_txs": {
    type: "line",
    title: "Send Txs",
    description: "Shows the median time taken for send txs",
    unit: "ns",
  },
  "latency/update_fork_choice": {
    type: "line",
    title: "Update Fork Choice",
    description: "Shows the median time taken for update fork choice",
    unit: "ns",
  },
  "latency/get_payload": {
    type: "line",
    title: "Get Payload",
    description: "Shows the median time taken for get payload",
    unit: "ns",
  },
  "latency/new_payload": {
    type: "line",
    title: "New Payload",
    description: "Shows the median time taken for new payload",
    unit: "ns",
  },
  "chain/inserts.50-percentile": {
    type: "line",
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
    title: "Account Updates",
    description:
      "Shows the median time taken for updating accounts during state validation",
    unit: "ns",
  },
  "chain/account/hashes.50-percentile": {
    type: "line",
    title: "Account Hashes",
    description:
      "Shows the median time taken for hashing accounts during state validation",
    unit: "ns",
  },
  "chain/storage/updates.50-percentile": {
    type: "line",
    title: "Storage Updates", // Renamed from 'Storage Writes' for consistency
    description:
      "Shows the median time taken for updating storage during state validation",
    unit: "ns",
  },
  "chain/validation.50-percentile": {
    type: "line",
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
    title: "Write (Misc)",
    description:
      "Shows the median time taken for miscellaneous block write operations (excluding commits)",
    unit: "ns",
  },
  "chain/account/commits.50-percentile": {
    type: "line",
    title: "Account Commits",
    description:
      "Shows the median time taken for committing account changes to the DB",
    unit: "ns",
  },
  "chain/storage/commits.50-percentile": {
    type: "line",
    title: "Storage Commits",
    description:
      "Shows the median time taken for committing storage changes to the DB",
    unit: "ns",
  },
  "chain/snapshot/commits.50-percentile": {
    type: "line",
    title: "Snapshot Commits",
    description:
      "Shows the median time taken for committing snapshot changes to the DB",
    unit: "ns",
  },
  "chain/triedb/commits.50-percentile": {
    type: "line",
    title: "TrieDB Commits",
    description: "Shows the median time taken for committing TrieDB changes",
    unit: "ns",
  },
  "transactions/per_block": {
    type: "line",
    title: "Transactions per Block",
    description: "Shows the number of transactions per block",
    unit: "count",
  },
  "gas/per_block": {
    type: "line",
    title: "Gas Per Block",
    description: "Shows the median gas per block",
    unit: "gas",
  },
};
