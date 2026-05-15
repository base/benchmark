import { Navigate, Route, Routes } from "react-router-dom";
import RunIndex from "./pages/RunIndex";
import RunComparison from "./pages/RunComparison";
import RedirectToLatestRun from "./pages/RedirectToLatestRun";
import LoadTestLanding from "./pages/LoadTestLanding";
import LoadTestAllRuns from "./pages/LoadTestAllRuns";
import LoadTestDetail from "./pages/LoadTestDetail";
import BenchmarkLoadTestDetail from "./pages/BenchmarkLoadTestDetail";
import ErrorBoundary from "./components/ErrorBoundary";

function App() {
  return (
    <ErrorBoundary>
      <Routes>
        <Route path="/" element={<RedirectToLatestRun />} />
        <Route
          path="/load-tests"
          element={<Navigate to="/load-tests/sepolia" replace />}
        />
        <Route path="/load-tests/:network" element={<LoadTestLanding />} />
        <Route path="/load-tests/:network/all" element={<LoadTestAllRuns />} />
        <Route
          path="/load-tests/:network/:timestamp"
          element={<LoadTestDetail />}
        />
        <Route
          path="/benchmark-load-test/:outputDir"
          element={<BenchmarkLoadTestDetail />}
        />
        <Route path="/:benchmarkRunId" element={<RunIndex />} />
        <Route
          path="/run-comparison/:benchmarkRunId"
          element={<RunComparison />}
        />
      </Routes>
    </ErrorBoundary>
  );
}

export default App;
