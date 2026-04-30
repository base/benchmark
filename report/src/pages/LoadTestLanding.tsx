import { Navigate, useParams } from "react-router-dom";
import Navbar from "../components/Navbar";
import { useLoadTestList } from "../utils/useDataSeries";

const DEFAULT_NETWORK = "sepolia";

const LoadTestLanding = () => {
  const { network = DEFAULT_NETWORK } = useParams();
  const { data: entries, isLoading, error } = useLoadTestList(network);

  if (!isLoading && !error && entries && entries.length > 0) {
    // List endpoint returns runs sorted newest-first; take entry 0 as latest.
    const latest = entries[0];
    return (
      <Navigate
        to={`/load-tests/${latest.network}/${latest.timestamp}`}
        replace
      />
    );
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <Navbar />
      <main className="px-8 py-6 max-w-5xl mx-auto">
        <header className="mb-6">
          <h1 className="text-2xl font-semibold text-slate-900">Load Tests</h1>
          <p className="text-sm text-slate-500 mt-1">
            Network: <span className="font-mono">{network}</span>
          </p>
        </header>

        {isLoading && (
          <div className="text-sm text-slate-500">Loading load tests…</div>
        )}

        {error && (
          <div className="border border-red-200 bg-red-50 text-red-800 rounded-lg p-4 text-sm">
            Failed to load load tests: {String(error)}
          </div>
        )}

        {!isLoading && !error && (!entries || entries.length === 0) && (
          <div className="border border-slate-200 bg-white rounded-lg p-8 text-center text-sm text-slate-500">
            No load test runs found for{" "}
            <span className="font-mono">{network}</span>.
          </div>
        )}
      </main>
    </div>
  );
};

export default LoadTestLanding;
