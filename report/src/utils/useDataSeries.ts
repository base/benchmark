import useSWR, { State, useSWRConfig } from "swr";
import { BenchmarkRuns, MetricData } from "../types";
import { useCallback } from "react";
import { getDataService } from "../services/dataService";

// Fetch metrics data using the configured data service
export const fetchMetrics = async (
  outputDir: string,
  nodeType: string,
): Promise<MetricData[]> => {
  const dataService = getDataService();
  return await dataService.getMetrics(outputDir, nodeType);
};

// Generate cache key for metrics
const metricsKey = (runId: string, outputDir: string, nodeType: string) => {
  return `metrics-${runId}-${outputDir}-${nodeType}`;
};

// Hook to fetch benchmark metadata using the configured data service
export const useTestMetadata = () => {
  const fetcher = useCallback(async (): Promise<BenchmarkRuns> => {
    const dataService = getDataService();
    return await dataService.getMetadata();
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
      const cachedData = cache.get(cacheKey) as State<MetricData[]> | undefined;

      if (cachedData?.data) {
        return cachedData.data;
      }

      // Fetch from API and cache
      const data = await mutate(cacheKey, async () => {
        return await fetchMetrics(outputDir, role);
      });

      if (!data) {
        throw new Error(
          `Failed to fetch data for ${runId}/${outputDir}/${role}`,
        );
      }

      return data;
    },
    [cache, mutate],
  );

  const multiFetcher = async (
    urlsToFetch: [runId: string, outputDir: string, role: string][],
  ) => {
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
