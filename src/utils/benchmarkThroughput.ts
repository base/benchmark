import { MetricData } from "../types";

const GAS_PER_BLOCK = "gas/per_block";

// The snapshot benchmark runner in base/base writes this per-block validator
// timing into the validator metric artifact.
export const VALIDATOR_NEW_PAYLOAD_TIME =
  "reth_consensus_engine_beacon_new_payload_latency_avg";

/**
 * Calculates throughput from the measured work and measured processing time:
 * sum(gas per block) / sum(processing time per block).
 */
export const gasPerSecondFromBlockMetrics = (
  blocks: MetricData[],
  processingTimeMetric: string,
): number | undefined => {
  let totalGas = 0;
  let totalProcessingTime = 0;

  for (const block of blocks) {
    const gas = block.ExecutionMetrics[GAS_PER_BLOCK];
    const processingTime = block.ExecutionMetrics[processingTimeMetric];

    if (
      typeof gas !== "number" ||
      typeof processingTime !== "number" ||
      !Number.isFinite(gas) ||
      !Number.isFinite(processingTime) ||
      gas < 0 ||
      processingTime <= 0
    ) {
      continue;
    }

    totalGas += gas;
    totalProcessingTime += processingTime;
  }

  return totalProcessingTime > 0 ? totalGas / totalProcessingTime : undefined;
};
