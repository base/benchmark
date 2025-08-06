import { Route, Routes } from "react-router-dom";
import RunIndex from "./pages/RunIndex";
import RunComparison from "./pages/RunComparison";
import RedirectToLatestRun from "./pages/RedirectToLatestRun";
import ErrorBoundary from "./components/ErrorBoundary";

function App() {
  return (
    <ErrorBoundary>
      <Routes>
        <Route path="/" element={<RedirectToLatestRun />} />
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
