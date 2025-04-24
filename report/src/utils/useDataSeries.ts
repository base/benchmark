import useSWR, { State, useSWRConfig } from "swr";
import { BenchmarkRuns, MetricData } from "../types";
import { useCallback } from "react";

export const fetchMetrics = async (
  outputDir: string,
  nodeType: string,
): Promise<MetricData[]> => {
  const response = await fetch(`output/${outputDir}/metrics-${nodeType}.json`);
  return await response.json();
};

const metricsKey = (outputDir: string, nodeType: string) => {
  return `output/${outputDir}/metrics-${nodeType}.json`;
};

export const useTestMetadata = () => {
  const fetcher = useCallback(async () => {
    const response = await fetch("output/test_metadata.json");
    const jsonData = await response.json();

    return jsonData as BenchmarkRuns;
  }, []);

  return useSWR("output/test_metadata.json", fetcher);
};

export const useMultipleDataSeries = (
  urlsToFetch: [outputDir: string, role: string][],
) => {
  const { cache, mutate } = useSWRConfig();

  const fetcher = useCallback(
    async (url: [outputDir: string, role: string]) => {
      const [outputDir, role] = url;

      const cachedData = cache.get(metricsKey(outputDir, role)) as
        | State<MetricData[]>
        | undefined;
      if (cachedData?.data) {
        return cachedData.data;
      }
      const data = await mutate(metricsKey(outputDir, role), async () => {
        const response = await fetchMetrics(outputDir, role);
        return response;
      });

      if (!data) {
        throw new Error(`Failed to fetch data for ${outputDir} and ${role}`);
      }
      return data;
    },
    [cache, mutate],
  );

  const multiFetcher = (urlsToFetch: [outputDir: string, role: string][]) => {
    return Promise.all(
      urlsToFetch.map((url) => {
        const [outputDir, role] = url;
        return fetcher([outputDir, role]);
      }),
    );
  };

  return useSWR(urlsToFetch, multiFetcher);
};
