// Unified data service that works with both static files and API servers
// Since the API now emulates the static file structure, we only need one service
import {
  BenchmarkRun,
  BenchmarkRuns,
  LoadTestEntry,
  LoadTestResult,
  MetricData,
} from "../types";

const networkForRun = (run: BenchmarkRun): string =>
  run.outputDir.split("-")[0] || `chain-${run.testConfig.ChainId ?? "unknown"}`;

const hasLoadTestArtifact = (run: BenchmarkRun): boolean =>
  Boolean(run.result?.artifacts?.loadTestResult);

export interface DataServiceConfig {
  baseUrl: string; // Base URL for both static and API modes
}

// Unified data service that works with both static files and API servers
export class DataService {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  async getMetadata(): Promise<BenchmarkRuns> {
    const response = await fetch(`${this.baseUrl}output/metadata.json`);

    if (!response.ok) {
      throw new Error(
        `Failed to fetch metadata: ${response.status} ${response.statusText}`,
      );
    }

    return await response.json();
  }

  async getMetrics(outputDir: string, nodeType: string): Promise<MetricData[]> {
    const metricsPath = `${this.baseUrl}output/${outputDir}/metrics-${nodeType}.json`;
    const response = await fetch(metricsPath);

    if (!response.ok) {
      throw new Error(
        `Failed to fetch metrics: ${response.status} ${response.statusText}`,
      );
    }

    return await response.json();
  }

  async getLoadTestList(network: string): Promise<LoadTestEntry[]> {
    const metadata = await this.getMetadata();
    return metadata.runs
      .filter(
        (run) => hasLoadTestArtifact(run) && networkForRun(run) === network,
      )
      .map((run) => ({
        network: networkForRun(run),
        outputDir: run.outputDir,
        createdAt: run.createdAt,
        testName: run.testName,
        transactionPayload:
          typeof run.testConfig.TransactionPayload === "string"
            ? run.testConfig.TransactionPayload
            : undefined,
        blockTimeMilliseconds: Number(run.testConfig.BlockTimeMilliseconds),
      }))
      .sort(
        (left, right) =>
          new Date(right.createdAt).getTime() -
          new Date(left.createdAt).getTime(),
      );
  }

  async getLoadTestResult(
    network: string,
    outputDir: string,
  ): Promise<LoadTestResult> {
    const metadata = await this.getMetadata();
    const run = metadata.runs.find(
      (candidate) =>
        candidate.outputDir === outputDir &&
        networkForRun(candidate) === network &&
        hasLoadTestArtifact(candidate),
    );
    if (!run) {
      throw new Error(
        `No load-test artifact found for ${network}/${outputDir}`,
      );
    }

    const filename = run.result?.artifacts?.loadTestResult;
    const response = await fetch(
      `${this.baseUrl}output/${encodeURIComponent(outputDir)}/${encodeURIComponent(filename ?? "load-test-result.json")}`,
    );

    if (!response.ok) {
      throw new Error(
        `Failed to fetch load test result: ${response.status} ${response.statusText}`,
      );
    }

    return await response.json();
  }
}

// Configuration helper to determine base URL from environment
export function getDataSourceConfig(): DataServiceConfig {
  // Check for environment variable (build-time or runtime)
  const getEnvVar = (key: string): string | undefined => {
    // Client-side: check for runtime configuration first
    if (typeof window !== "undefined") {
      const runtimeConfig = (window as unknown as Record<string, unknown>)
        .__RUNTIME_CONFIG__;
      if (
        runtimeConfig &&
        typeof runtimeConfig === "object" &&
        key in runtimeConfig
      ) {
        const value = (runtimeConfig as Record<string, unknown>)[key];
        return typeof value === "string" ? value : undefined;
      }
    }

    // Fallback to Vite environment variable
    const viteEnv = (import.meta as unknown as Record<string, unknown>).env;
    if (viteEnv && typeof viteEnv === "object") {
      const envVar = (viteEnv as Record<string, unknown>)[`VITE_${key}`];
      return typeof envVar === "string" ? envVar : undefined;
    }
    return undefined;
  };

  // Determine base URL based on configuration
  const apiBaseUrl = getEnvVar("API_BASE_URL");
  const dataSource = getEnvVar("DATA_SOURCE") || "static";

  if (dataSource === "api" && apiBaseUrl) {
    // API mode: use the configured API base URL (ensure trailing slash)
    return { baseUrl: apiBaseUrl.replace(/\/$/, "") + "/" };
  } else {
    // Static mode: use current origin (empty string means relative to current domain)
    return { baseUrl: "" };
  }
}

// Global data service instance
let dataServiceInstance: DataService | null = null;

export function getDataService(): DataService {
  if (!dataServiceInstance) {
    const config = getDataSourceConfig();
    dataServiceInstance = new DataService(config.baseUrl);
  }
  return dataServiceInstance;
}

// Allow resetting the service instance (useful for testing)
export function resetDataService(): void {
  dataServiceInstance = null;
}
