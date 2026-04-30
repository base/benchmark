import { Link, useParams } from "react-router-dom";
import Navbar from "../components/Navbar";
import { useLoadTestList } from "../utils/useDataSeries";
import { formatLoadTestTimestamp } from "../utils/formatters";

const DEFAULT_NETWORK = "sepolia";

const LoadTestAllRuns = () => {
  const { network = DEFAULT_NETWORK } = useParams();
  const { data: entries, isLoading, error } = useLoadTestList(network);

  return (
    <div className="min-h-screen bg-slate-50">
      <Navbar />
      <main className="px-8 py-6 max-w-5xl mx-auto">
        <header className="mb-6">
          <Link
            to={`/load-tests/${network}`}
            className="text-sm text-blue-600 hover:underline"
          >
            ← Back to latest run
          </Link>
          <h1 className="text-2xl font-semibold text-slate-900 mt-2">
            All load test runs
          </h1>
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

        {entries && entries.length > 0 && (
          <div className="border border-slate-200 bg-white rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 text-slate-500 text-xs uppercase tracking-wide">
                <tr>
                  <th className="text-left font-medium px-6 py-3">Run</th>
                  <th className="text-left font-medium px-6 py-3">Network</th>
                  <th className="text-right font-medium px-6 py-3">Details</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {entries.map((entry) => (
                  <tr
                    key={`${entry.network}-${entry.timestamp}`}
                    className="hover:bg-slate-50"
                  >
                    <td className="px-6 py-3 font-mono text-slate-900">
                      <Link
                        to={`/load-tests/${entry.network}/${entry.timestamp}`}
                        className="hover:underline"
                      >
                        {formatLoadTestTimestamp(entry.timestamp)}
                      </Link>
                      <div className="text-xs text-slate-400 font-mono mt-0.5">
                        {entry.timestamp}
                      </div>
                    </td>
                    <td className="px-6 py-3 text-slate-600">
                      {entry.network}
                    </td>
                    <td className="px-6 py-3 text-right">
                      <Link
                        to={`/load-tests/${entry.network}/${entry.timestamp}`}
                        className="text-blue-600 hover:underline"
                      >
                        View →
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </div>
  );
};

export default LoadTestAllRuns;
