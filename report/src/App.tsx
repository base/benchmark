import { useMemo, useState } from "react";
import ChartSelector, { DataFileRequest } from "./components/ChartSelector";
import ChartGrid from "./components/ChartGrid";
import { useTestMetadata, useMultipleDataSeries } from "./utils/useDataSeries";

function App() {
  const [dataQuery, setDataQuery] = useState<DataFileRequest[]>([]);

  const { data: benchmarkRuns, isLoading: isLoadingBenchmarkRuns } =
    useTestMetadata();

  const dataQueryKey = useMemo(() => {
    return dataQuery.map(
      (query) => [query.outputDir, query.role] as [string, string],
    );
  }, [dataQuery]);

  const { data: dataPerFile, isLoading } = useMultipleDataSeries(dataQueryKey);
  const data = useMemo(() => {
    if (!dataPerFile) {
      return dataPerFile;
    }

    return dataPerFile.map((data, index) => {
      const { name } = dataQuery[index];
      return {
        name,
        data,
      };
    });
  }, [dataPerFile, dataQuery]);

  if (!benchmarkRuns || isLoadingBenchmarkRuns) {
    return <div>Loading...</div>;
  }

  return (
    <div className="container">
      <h1>Base Bench Metrics</h1>
      <ChartSelector
        onChangeDataQuery={setDataQuery}
        benchmarkRuns={benchmarkRuns}
      />
      {isLoading ? "Loading..." : <ChartGrid data={data ?? []} />}
    </div>
  );
}

export default App;
