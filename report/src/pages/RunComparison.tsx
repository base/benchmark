import { useMemo, useState } from "react";
import ChartSelector, {
  DataSelection,
  EmptyDataSelection,
} from "../components/ChartSelector";
import ChartGrid from "../components/ChartGrid";
import { useTestMetadata, useMultipleDataSeries } from "../utils/useDataSeries";
import { DataSeries } from "../types";
import { useParams } from "react-router-dom";
import Navbar from "../components/Navbar";

function RunComparison() {
  let { benchmarkRunId } = useParams();

  if (!benchmarkRunId) {
    throw new Error("Benchmark run ID is required");
  }

  const [selection, setSelection] = useState<DataSelection>(EmptyDataSelection);

  const { data: allBenchmarkRuns, isLoading: isLoadingBenchmarkRuns } =
    useTestMetadata();

  const latestBenchmarkRun = useMemo(() => {
    return allBenchmarkRuns?.runs.sort(
      (a, b) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
    )[0];
  }, [allBenchmarkRuns]);

  if (latestBenchmarkRun && benchmarkRunId === "latest") {
    benchmarkRunId = `${latestBenchmarkRun.testConfig.BenchmarkRun}`;
  }

  const benchmarkRuns = useMemo(() => {
    return {
      runs:
        allBenchmarkRuns?.runs.filter(
          (run) =>
            run.testConfig.BenchmarkRun === benchmarkRunId &&
            run.result?.complete &&
            run.result.success,
        ) ?? [],
    };
  }, [allBenchmarkRuns, benchmarkRunId]);

  const dataQueryKey = useMemo(() => {
    return selection.data.map((query) => {
      // Find the run that matches this outputDir to get the runId
      const run = benchmarkRuns.runs.find(
        (r) => r.outputDir === query.outputDir,
      );
      const runId = run?.id || query.outputDir; // Fallback to outputDir if no ID found
      return [runId, query.outputDir, query.role] as [string, string, string];
    });
  }, [selection.data, benchmarkRuns]);

  const { data: dataPerFile, isLoading } = useMultipleDataSeries(dataQueryKey);
  const data = useMemo(() => {
    if (!dataPerFile) {
      return dataPerFile;
    }

    return dataPerFile.map((data, index): DataSeries => {
      const { name, color } = selection.data[index];
      return {
        name,
        data,
        color,
        thresholds: selection.data[index].thresholds,
      };
    });
  }, [dataPerFile, selection.data]);

  if (!benchmarkRuns || isLoadingBenchmarkRuns) {
    return <div>Loading...</div>;
  }

  return (
    <div className="flex flex-col w-full min-h-screen">
      <Navbar urlPrefix="/run-comparison" />
      <div className="flex flex-col w-full flex-grow">
        <div className="p-8">
          <ChartSelector
            onChangeDataQuery={setSelection}
            benchmarkRuns={benchmarkRuns}
          />
          {isLoading ? (
            "Loading..."
          ) : (
            <ChartGrid role={selection.role} data={data ?? []} />
          )}
        </div>
      </div>
    </div>
  );
}

export default RunComparison;
