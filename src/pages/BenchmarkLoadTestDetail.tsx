import { useParams } from "react-router-dom";
import Navbar from "../components/Navbar";
import { useBenchmarkLoadTestResult } from "../utils/useDataSeries";
import { LoadTestReportContent } from "./LoadTestDetail";

const BenchmarkLoadTestDetail = () => {
  const { outputDir } = useParams();
  const {
    data: result,
    isLoading,
    error,
  } = useBenchmarkLoadTestResult(outputDir);

  return (
    <div className="min-h-screen bg-slate-50">
      <Navbar />
      <main className="px-8 py-6 max-w-5xl mx-auto flex flex-col gap-y-6">
        {isLoading && (
          <div className="text-sm text-slate-500">
            Loading benchmark load test…
          </div>
        )}

        {error && (
          <div className="border border-red-200 bg-red-50 text-red-800 rounded-lg p-4 text-sm">
            Failed to load benchmark load test result: {String(error)}
          </div>
        )}

        {result && (
          <LoadTestReportContent
            result={result}
            title="Benchmark load test"
            subtitle={
              <>
                Output dir: <span className="font-mono">{outputDir}</span>
              </>
            }
            backLink={{
              to: "/latest",
              label: "View benchmark runs →",
            }}
          />
        )}
      </main>
    </div>
  );
};

export default BenchmarkLoadTestDetail;
