export interface MetricData {
  BlockNumber: number;
  Timestamp?: string | number;
  ElapsedMilliseconds?: number;
  ExecutionMetrics: {
    [key: string]: number;
  };
}

export interface DataSeries {
  data: MetricData[];
  name: string;
  color?: string;
  thresholds?: {
    warning?: Record<string, number>;
    error?: Record<string, number>;
  };
}

export interface ChartDimensions {
  width: number;
  height: number;
  margin: {
    top: number;
    right: number;
    bottom: number;
    left: number;
  };
}

// Interface for programmatic chart creation (if used elsewhere)
export interface ChartOptions {
  container: HTMLElement;
  series: DataSeries[];
  metricKey: string;
  title?: string;
  description?: string;
}

// Define the structure for chart configuration entries from the manifest
export interface ChartConfig {
  title: string;
  description: string;
  type: "line";
  unit?:
    | "ns"
    | "us"
    | "ms"
    | "s"
    | "bytes"
    | "gas"
    | "count"
    | "gas/s"
    | "blocks";
  aliases?: string[];
}

export interface MachineInfo {
  type?: string; // e.g., i4i.32xlarge
  provider?: string; // aws or gcp
  region?: string; // e.g., us-east-1
  fileSystem?: string; // e.g., ext4
}

export interface BenchmarkRun {
  id: string;
  sourceFile: string;
  testName: string;
  testDescription: string;
  outputDir: string;
  createdAt: string;
  testConfig: Record<string, string | number>;
  machineInfo?: MachineInfo;
  thresholds?: {
    warning?: Record<string, number>;
    error?: Record<string, number>;
  };
  result: {
    success: boolean;
    complete?: boolean;
    clientVersion?: string;
    sequencerMetrics?: {
      gasPerSecond: number;
      forkChoiceUpdated: number;
      getPayload: number;
      sendTxs?: number;
    };
    validatorMetrics?: {
      gasPerSecond: number;
      newPayload: number;
    };
  } | null;
}

export interface BenchmarkRuns {
  runs: BenchmarkRun[];
}

export type RunStatus =
  | "incomplete"
  | "success"
  | "fatal"
  | "error"
  | "warning";

const statusRelatedMetrics = {
  "latency/fork_choice_updated": ["forkChoiceUpdated", "sequencer", 1e9],
  "latency/get_payload": ["getPayload", "sequencer", 1e9],
  "latency/new_payload": ["newPayload", "validator", 1e9],
} as const;

export type BenchmarkRunWithStatus = BenchmarkRun & { status: RunStatus };

// -----------------------------------------------------------------------------
// Load tests
// -----------------------------------------------------------------------------
//
// These types mirror the JSON written by the `base-load-tester` Rust binary and
// served by report-api at:
//   GET /api/v1/load-tests/:network                  -> LoadTestEntry[]
//   GET /api/v1/load-tests/:network/:timestamp       -> LoadTestResult
//
// IMPORTANT: types are hand-maintained. If the Rust schema changes, update here.
// See `upgrades.md` for planned producer-side changes (schema_version, metadata
// block, sidecar index, etc.) that would let us generate these instead.

/**
 * Mirrors Rust `std::time::Duration` as serialized by serde_json.
 */
export interface RustDuration {
  secs: number;
  nanos: number;
}

// Mirrors Rust `LatencyMetrics`. No `count` field — only
// `FlashblocksLatencyStats` carries a sample count.
export interface LatencyStats {
  min: RustDuration;
  max: RustDuration;
  mean: RustDuration;
  p50: RustDuration;
  p95: RustDuration;
  p99: RustDuration;
}

export interface FlashblocksLatencyStats extends LatencyStats {
  count: number;
  p90: RustDuration;
}

export interface ThroughputStats {
  total_submitted: number;
  total_confirmed: number;
  total_failed: number;
  total_reverted: number;
  tps: number;
  gps: number;
  duration: RustDuration;
}

export interface ThroughputPercentiles {
  tps_p50: number;
  tps_p90: number;
  tps_p99: number;
  tps_max: number;
  gps_p50: number;
  gps_p90: number;
  gps_p99: number;
  gps_max: number;
}

export interface BlockRange {
  // Omitted when no test transactions were confirmed.
  first_block?: number;
  last_block?: number;
  block_count: number;
}

export interface GasStats {
  total_gas: number;
  avg_gas: number;
  // WARNING: This can exceed Number.MAX_SAFE_INTEGER (2^53). The Rust binary
  // currently emits it as a JSON number, which silently loses precision in
  // JavaScript. See upgrades.md P0 #1 — once backend stringifies it, change
  // the type here to `string` and parse with BigInt at the formatting boundary.
  total_cost_wei: number;
  avg_gas_price: number;
}

// Producer emits this as a JSON tuple `[reason, count]`, not an object.
// Once upgrades.md P3 #7 lands and the producer switches to `{reason, count}`
// objects, change the type here and update the page accessor.
export type FailureReason = [string, number];

/**
 * Run parameters captured by the producer at start time. All fields mirror the
 * Rust config struct verbatim; the page omits any null-valued field rather than
 * showing "—" so older runs (which lack `config` entirely) and runs with mixed
 * nulls present a uniform UI.
 */
export interface LoadTestConfig {
  funding_amount: string;
  sender_count: number;
  sender_offset: number;
  in_flight_per_sender: number;
  // Closed-loop producer fields. Open-loop (base#30092f0) dropped these in
  // favor of funding_batch_size — treat as optional so both schemas render.
  batch_size?: number | null;
  batch_timeout?: string | null;
  // Open-loop / funding-phase RPC batch size. Absent on older closed-loop runs.
  funding_batch_size?: number | null;
  // Option in Rust ConfigSummary — null when unset; may be absent on older runs.
  duration: string | null;
  target_gps?: number | null;
  // Open-loop adaptive pacing ceiling; prefer when target_gps is absent.
  max_target_gps?: number | null;
  seed: number;
  chain_id: number | null;
  // Storage-type entries also carry `contract` and `slots_per_tx` (flattened
  // from the Rust `TxTypeConfig::Storage` variant via `#[serde(tag = "type")]`).
  transactions: Array<{
    type: string;
    weight: number;
    contract?: string;
    slots_per_tx?: number;
  }>;
  fresh_recipient_ratio?: number;
  looper_contract: string | null;
  swap_token_amount: string;
  // Present on newer producers; unused in the UI today.
  b20_mint_amount?: string;
  // Producer omits unless real-token setup is enabled; loosely typed.
  real_token_setup?: Record<string, unknown> | null;
}

/**
 * One sample of the throughput timeseries. Producer emits one sample per
 * window (≈0.5–1s apart, irregular). Plot against `elapsed_secs` directly,
 * not array index, so the curve stays time-accurate.
 */
export interface ThroughputSample {
  elapsed_secs: number;
  tps: number;
  gps: number;
}

// Mirrors Rust `MetricsSummary` (base/base#3695). Reports client-to-client
// end-to-end latency over the full run; the earlier observed-window/tail split
// and per-tx block_receipt_delay were removed.
export interface LoadTestResult {
  // Present only on runs where a fatal error stopped the test.
  error?: string;
  block_latency: LatencyStats;
  flashblocks_latency: FlashblocksLatencyStats;
  throughput: ThroughputStats;
  throughput_percentiles: ThroughputPercentiles;
  throughput_timeseries: ThroughputSample[];
  gas: GasStats;
  block_range: BlockRange;
  // Element type is best-effort until upgrades.md P3 #7 lands. Empty arrays
  // dominate today, so we have no live samples to verify against.
  top_failure_reasons: FailureReason[];
  // Optional for back-compat: older S3 runs predate this field, and the page
  // omits the config card when absent.
  config?: LoadTestConfig;
}

/**
 * One entry in the list returned by `GET /api/v1/load-tests/:network`.
 * Backend sorts newest-first by timestamp string (lexicographic over the
 * "YYYY-MM-DD-HH-MM-SS" format works because all components are zero-padded).
 */
export interface LoadTestEntry {
  network: string;
  timestamp: string;
}

export const getTestRunsWithStatus = (
  runs: BenchmarkRuns,
): BenchmarkRunWithStatus[] => {
  return runs.runs.map((run) => {
    if (!run.result?.complete) {
      return { ...run, status: "incomplete" as RunStatus };
    }
    if (!run.result?.success) {
      return { ...run, status: "error" as RunStatus };
    }
    const warnThresholds = run.thresholds?.warning;
    const errorThresholds = run.thresholds?.error;

    const checkThresholds = (
      level: "warning" | "error",
      thresholds: Record<string, number>,
    ): RunStatus | undefined => {
      for (const [metric, threshold] of Object.entries(thresholds)) {
        const [statusThresholdName, statusType, scale] =
          statusRelatedMetrics[metric as keyof typeof statusRelatedMetrics] ??
          [];
        if (!statusThresholdName || !statusType || !scale) {
          // metrics not related to a summary stat are not considered for status
          continue;
        }

        const metricsName = `${statusType}Metrics` as const;

        // cast to never to avoid type errors here - if an error occurs, check statusRelatedMetrics
        const value = run.result?.[metricsName]?.[statusThresholdName as never];
        if (typeof value !== "number") {
          // non-numbers and undefined values are skipped
          continue;
        }
        if (value * scale > threshold) {
          return level;
        }
      }
    };

    if (errorThresholds) {
      const errorStatus = checkThresholds("error", errorThresholds);
      if (errorStatus) {
        return { ...run, status: errorStatus };
      }
    }

    if (warnThresholds) {
      const warnStatus = checkThresholds("warning", warnThresholds);
      if (warnStatus) {
        return { ...run, status: warnStatus };
      }
    }

    return { ...run, status: "success" as RunStatus };
  });
};
