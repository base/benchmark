import { useTestMetadata } from "../utils/useDataSeries";

function RunIndex() {
  const { data: benchmarkRuns, isLoading: isLoadingBenchmarkRuns } =
    useTestMetadata();


  if (!benchmarkRuns || isLoadingBenchmarkRuns) {
    return <div>Loading...</div>;
  }

  return (
    <div className="container">
      {
        benchmarkRuns?.runs.map((run) => (
          <div key={run.outputDir}>
            <h2>{run.name}</h2>
            <p>Output Directory: {run.outputDir}</p>
            <p>Role: {run.role}</p>
            <p>Test Name: {run.testName}</p>
            <p>Test Type: {run.testType}</p>
          </div>
        ))
      }
    </div>
  );
}

export default RunIndex;
