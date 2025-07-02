export interface MetricData {
  BlockNumber: number;
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
    | "blocks"; // Add 'gas/s', ensure 's' is present
}

export interface BenchmarkRun {
  id: string;
  sourceFile: string;
  testName: string;
  testDescription: string;
  outputDir: string;
  createdAt: string;
  testConfig: Record<string, string | number>;
  thresholds?: {
    warning?: Record<string, number>;
    error?: Record<string, number>;
  };
  result: {
    success: boolean;
    complete?: boolean;
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
