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
  unit?: "ns" | "us" | "ms" | "s" | "bytes" | "gas" | "count" | "gas/s"; // Add 'gas/s', ensure 's' is present
}

export interface BenchmarkRun {
  sourceFile: string;
  testName: string;
  testDescription: string;
  outputDir: string;
  testConfig: Record<string, string | number>;
  result: {
    success: boolean;
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
  };
}

export interface BenchmarkRuns {
  runs: BenchmarkRun[];
  createdAt: string;
}
