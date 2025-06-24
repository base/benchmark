import { useMemo, useState } from "react";
import ChartSelector, { SelectedData } from "../components/ChartSelector";
import ChartGrid from "../components/ChartGrid";
import { useTestMetadata, useMultipleDataSeries } from "../utils/useDataSeries";
import { DataSeries } from "../types";

function RunComparison() {
  const [selection, setSelection] = useState<SelectedData[]>([]);

  const { data: benchmarkRuns, isLoading: isLoadingBenchmarkRuns } =
    useTestMetadata();

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
    <div className="p-8">
      <ChartSelector
        onChangeDataQuery={setSelection}
        benchmarkRuns={benchmarkRuns}
      />
      {isLoading ? "Loading..." : <ChartGrid data={data ?? []} />}
    </div>
  );
}

export default RunComparison;
