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
  type: 'line';
  unit?: 'ns' | 'us' | 'ms' | 's' | 'bytes' | 'gas' | 'count'; // Add 'us', add more units as needed
}

export interface BenchmarkRun {
  sourceFile: string;
  testName: string;
  testDescription: string;
  outputDir: string;
  testConfig: Record<string, string | number>;
}

export interface BenchmarkRuns {
  runs: BenchmarkRun[];
}

export function getBenchmarkVariables(runs: BenchmarkRun[]) {
  const inferredConfig: Record<string, Array<string | number | boolean>> = {};

  for (const run of runs) {
    for (const [key, value] of Object.entries(run.testConfig)) {
      if (!inferredConfig[key]) {
        inferredConfig[key] = [];
      }
      inferredConfig[key].push(value);
    }
  }

  return Object.fromEntries(
    Object.entries(inferredConfig).filter(([_, values]) => values.length > 1).map(([key, values]) => {
      return [key, [...new Set(values)]];
    })
  );
}

