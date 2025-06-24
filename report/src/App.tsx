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
            <Link to="/">
              <img src={Logo} className="w-8 h-8" />
            </Link>
            <div className="font-medium">Client Benchmark Report</div>
          </div>
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
      <Routes>
        <Route path="/" element={<RunIndex />} />
        <Route path="/run-comparison" element={<RunComparison />} />
      </Routes>
    </>
  );
}

export default App;
