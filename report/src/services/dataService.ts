// Unified data service that works with both static files and API servers
// Since the API now emulates the static file structure, we only need one service
import { BenchmarkRuns, MetricData } from "../types";

export interface DataServiceConfig {
  baseUrl: string; // Base URL for both static and API modes
}

// Unified data service that works with both static files and API servers
export class DataService {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl.replace(/\/$/, ""); // Remove trailing slash
  }

  async getMetadata(): Promise<BenchmarkRuns> {
    const response = await fetch(`${this.baseUrl}/output/metadata.json`);

    if (!response.ok) {
      throw new Error(
        `Failed to fetch metadata: ${response.status} ${response.statusText}`,
      );
    }

    return await response.json();
  }

  async getMetrics(outputDir: string, nodeType: string): Promise<MetricData[]> {
    const metricsPath = `${this.baseUrl}/output/${outputDir}/metrics-${nodeType}.json`;
    const response = await fetch(metricsPath);

    if (!response.ok) {
      throw new Error(
        `Failed to fetch metrics: ${response.status} ${response.statusText}`,
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
    // API mode: use the configured API base URL
    return { baseUrl: apiBaseUrl };
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
