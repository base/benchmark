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
  type: 'line' | 'bar';
  unit?: 'ns' | 'us' | 'ms' | 's' | 'bytes' | 'gas' | 'count'; // Add 'us', add more units as needed
}
