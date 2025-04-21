import { Link, Route, Routes } from "react-router-dom";
import RunIndex from "./pages/RunIndex";
import RunComparison from "./pages/RunComparison";
import Logo from "./assets/logo.svg";
import { useTestMetadata } from "./utils/useDataSeries";

function App() {
  const { data: benchmarkRuns } = useTestMetadata();

  const benchmarkTime = benchmarkRuns?.createdAt
    ? new Date(benchmarkRuns.createdAt)
    : null;

  return (
    <>
      <nav className="flex px-8 border-b border-slate-300 items-center bg-white gap-x-4">
        <div className="flex items-center gap-x-4 flex-grow">
          <div className="flex items-center gap-x-4 py-4">
            <img src={Logo} className="w-8 h-8" />
            <div className="font-medium">Client Benchmark Report</div>
          </div>
          <Link to="/" className="text-slate-500 hover:text-slate-700 p-4">
            All Tests
          </Link>
        </div>
        <div>
          Showing Benchmark from{" "}
          {benchmarkTime ? (
            Intl.DateTimeFormat().format(benchmarkTime)
          ) : (
            <span className="text-slate-600">loading...</span>
          )}
        </div>
      </nav>
      <div className="p-8">
        <Routes>
          <Route path="/" element={<RunIndex />} />
          <Route path="/run-comparison" element={<RunComparison />} />
        </Routes>
      </div>
    </>
  );
}

export default App;
