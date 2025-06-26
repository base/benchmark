import { useMemo, useState } from "react";
import ChartSelector, { SelectedData } from "../components/ChartSelector";
import ChartGrid from "../components/ChartGrid";
import { useTestMetadata, useMultipleDataSeries } from "../utils/useDataSeries";
import { DataSeries } from "../types";
import { useParams } from "react-router-dom";
import Navbar from "../components/Navbar";
import BenchmarkRunDetails from "../components/BenchmarkRunDetails";

function RunComparison() {
  let { benchmarkRunId } = useParams();

  if (!benchmarkRunId) {
    throw new Error("Benchmark run ID is required");
  }

  const [selection, setSelection] = useState<SelectedData[]>([]);

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
    return selection.map(
      (query) => [query.outputDir, query.role] as [string, string],
    );
  }, [selection]);

  const { data: dataPerFile, isLoading } = useMultipleDataSeries(dataQueryKey);
  const data = useMemo(() => {
    if (!dataPerFile) {
      return dataPerFile;
    }

    return dataPerFile.map((data, index): DataSeries => {
      const { name, color } = selection[index];
      return {
        name,
        data,
        color,
        thresholds: selection[index].thresholds,
      };
    });
  }, [dataPerFile, selection]);

  if (!benchmarkRuns || isLoadingBenchmarkRuns) {
    return <div>Loading...</div>;
  }

  return (
    <div className="flex flex-col w-full min-h-screen">
      <Navbar urlPrefix="/run-comparison" />
      <div className="flex flex-col w-full flex-grow">
        <div className="p-8">
          <BenchmarkRunDetails benchmarkRuns={benchmarkRuns.runs} />
          <ChartSelector
            onChangeDataQuery={setSelection}
            benchmarkRuns={benchmarkRuns}
          />
          {isLoading ? "Loading..." : <ChartGrid data={data ?? []} />}
        </div>
      </div>
    </div>
  );
}

export default RunComparison;
