import { Link, Route, Routes } from "react-router-dom";
import RunIndex from "./pages/RunIndex";
import RunComparison from "./pages/RunComparison";
import Logo from "./assets/logo.svg";
import { useTestMetadata } from "./utils/useDataSeries";
import GithubIcon from "./assets/github.svg";
import FAQs from "./pages/FAQs";

function App() {
  const { data: benchmarkRuns } = useTestMetadata();

  const benchmarkTime = benchmarkRuns?.createdAt
    ? new Date(benchmarkRuns.createdAt)
    : null;

  return (
    <>
      <nav className="flex px-8 border-b border-slate-200 items-center bg-white gap-x-4">
        <div className="flex items-center gap-x-4 flex-grow">
          <div className="flex items-center gap-x-4 py-4">
            <Link to="/">
              <img src={Logo} className="w-8 h-8" />
            </Link>
            <div className="font-medium">Client Benchmark Report</div>
            <Link to="/faqs" className="text-sm text-slate-500">
              FAQs
            </Link>
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
        <div>
          <a
            target="_blank"
            rel="noreferrer noopener"
            href="https://github.com/base/benchmark"
            className="p-2"
          >
            <img src={GithubIcon} className="w-6 h-6" />
          </a>
        </div>
      </nav>
      <div className="p-8">
        <Routes>
          <Route path="/" element={<RunIndex />} />
          <Route path="/run-comparison" element={<RunComparison />} />
          <Route path="/faqs" element={<FAQs />} />
        </Routes>
      </div>
    </>
  );
}

export default App;
