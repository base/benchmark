import useSWR, { State, useSWRConfig } from "swr";
import { BenchmarkRuns, MetricData } from "../types";
import { useCallback } from "react";
import { apiUrls } from "../config/api";

// Generic fetcher function for API calls
const apiFetcher = async <T>(url: string): Promise<T> => {
  const response = await fetch(url);
  
  if (!response.ok) {
    throw new Error(`API request failed: ${response.status} ${response.statusText}`);
  }
  
  return await response.json();
};

// Fetch metrics data from S3 backend
export const fetchMetrics = async (
  runId: string,
  outputDir: string,
  nodeType: string,
): Promise<MetricData[]> => {
  const url = apiUrls.metrics(runId, outputDir, nodeType);
  return apiFetcher<MetricData[]>(url);
};

// Generate cache key for metrics
const metricsKey = (runId: string, outputDir: string, nodeType: string) => {
  return `metrics-${runId}-${outputDir}-${nodeType}`;
};

// Hook to fetch benchmark metadata from S3 backend
export const useTestMetadata = () => {
  const fetcher = useCallback(async (): Promise<BenchmarkRuns> => {
    const url = apiUrls.metadata();
    return apiFetcher<BenchmarkRuns>(url);
  }, []);

  return useSWR("benchmark-metadata", fetcher, {
    // Cache for 5 minutes since metadata doesn't change frequently
    dedupingInterval: 5 * 60 * 1000,
    // Revalidate on window focus to get latest data
    revalidateOnFocus: true,
    // Retry on error
    errorRetryCount: 3,
    errorRetryInterval: 5000,
  });
};

// Hook to fetch multiple data series from S3 backend
export const useMultipleDataSeries = (
  urlsToFetch: [runId: string, outputDir: string, role: string][],
) => {
  const { cache, mutate } = useSWRConfig();

  const fetcher = useCallback(
    async (url: [runId: string, outputDir: string, role: string]) => {
      const [runId, outputDir, role] = url;

      // Check cache first
      const cacheKey = metricsKey(runId, outputDir, role);
      const cachedData = cache.get(cacheKey) as
        | State<MetricData[]>
        | undefined;
      
      if (cachedData?.data) {
        return cachedData.data;
      }

      // Fetch from API and cache
      const data = await mutate(cacheKey, async () => {
        return await fetchMetrics(runId, outputDir, role);
      });

      if (!data) {
        throw new Error(`Failed to fetch data for ${runId}/${outputDir}/${role}`);
      }
      
      return data;
    },
    [cache, mutate],
  );

  const multiFetcher = async (urlsToFetch: [runId: string, outputDir: string, role: string][]) => {
    // Fetch all metrics in parallel
    const promises = urlsToFetch.map((url) => {
      const [runId, outputDir, role] = url;
      return fetcher([runId, outputDir, role]);
    });

    return Promise.all(promises);
  };

  return useSWR(urlsToFetch, multiFetcher, {
    // Cache for 1 hour since metrics data is static once generated
    dedupingInterval: 60 * 60 * 1000,
    // Don't revalidate on window focus since metrics don't change
    revalidateOnFocus: false,
    // Retry on error
    errorRetryCount: 3,
    errorRetryInterval: 5000,
  });
};

// Hook to check API health (useful for monitoring)
export const useApiHealth = () => {
  const fetcher = useCallback(async () => {
    const url = apiUrls.health();
    return apiFetcher<{ status: string; timestamp: string; service: string }>(url);
  }, []);

  return useSWR("api-health", fetcher, {
    refreshInterval: 30000, // Check every 30 seconds
    errorRetryCount: 2,
    errorRetryInterval: 10000,
  });
};
