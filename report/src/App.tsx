import { Route, Routes } from "react-router-dom";
import RunIndex from "./pages/RunIndex";
import RunComparison from "./pages/RunComparison";
import RedirectToLatestRun from "./pages/RedirectToLatestRun";

function App() {
  return (
    <Routes>
      <Route path="/" element={<RedirectToLatestRun />} />
      <Route path="/:benchmarkRunId" element={<RunIndex />} />
      <Route
        path="/run-comparison/:benchmarkRunId"
        element={<RunComparison />}
      />
    </Routes>
  );
}

export default App;
