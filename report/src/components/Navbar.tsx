import {
  Link,
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";
import clsx from "clsx";
import Logo from "../assets/logo.svg";
import { useLoadTestList, useTestMetadata } from "../utils/useDataSeries";
import { useCallback, useMemo } from "react";
import { uniqBy } from "lodash";
import {} from "react-router-dom";
import Select from "./Select";
import { formatLoadTestTimestamp } from "../utils/formatters";

interface ProvidedProps {
  urlPrefix?: string;
}

const DEFAULT_LOAD_TEST_NETWORK = "sepolia";

const Navbar = ({ urlPrefix }: ProvidedProps) => {
  const location = useLocation();
  const isLoadTestsRoute = location.pathname.startsWith("/load-tests");

  const { data: allBenchmarkRuns, isLoading } = useTestMetadata();

  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  const navigateToBenchmarkRun = useCallback(
    (benchmarkRunId: string) => {
      navigate({
        pathname: `${urlPrefix ?? ""}/${benchmarkRunId}`,
        search: searchParams?.toString() ?? undefined,
      });
    },
    [urlPrefix, searchParams, navigate],
  );

  const {
    benchmarkRunId,
    network: loadTestNetwork,
    timestamp: loadTestTimestamp,
  } = useParams();

  const activeLoadTestNetwork = loadTestNetwork ?? DEFAULT_LOAD_TEST_NETWORK;

  const { data: loadTestEntries, isLoading: isLoadingLoadTests } =
    useLoadTestList(isLoadTestsRoute ? activeLoadTestNetwork : null);

  const navigateToLoadTestRun = useCallback(
    (timestamp: string) => {
      navigate({
        pathname: `/load-tests/${activeLoadTestNetwork}/${timestamp}`,
      });
    },
    [activeLoadTestNetwork, navigate],
  );

  const loadTestOptions = useMemo(() => {
    if (!loadTestEntries) return [];
    return loadTestEntries.map((entry) => ({
      label: formatLoadTestTimestamp(entry.timestamp),
      value: entry.timestamp,
    }));
  }, [loadTestEntries]);

  const latestRun = useMemo(() => {
    return allBenchmarkRuns?.runs.sort(
      (a, b) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
    )[0];
  }, [allBenchmarkRuns]);

  const benchmarkRunOptions = useMemo(() => {
    const options = allBenchmarkRuns?.runs.map((run) => {
      return {
        label: `${run.testName} - ${Intl.DateTimeFormat("en-US", {
          dateStyle: "short",
          timeStyle: "short",
        }).format(new Date(run.createdAt))}`,
        value: run.testConfig.BenchmarkRun,
        benchmarkRunId: run.testConfig.BenchmarkRun,
      };
    });

    const uniqueOptions = uniqBy(options, "value");

    if (latestRun) {
      uniqueOptions.unshift({
        label: `Latest - ${latestRun.testName}`,
        value: "latest",
        benchmarkRunId: latestRun.testConfig.BenchmarkRun,
      });
    }

    const optionsWithTestNum = uniqueOptions.map((option) => {
      const allRunsMatching = allBenchmarkRuns?.runs.filter(
        (r) => r.testConfig.BenchmarkRun === option.benchmarkRunId,
      );

      const numSuccess = allRunsMatching?.filter(
        (r) => r.result?.complete && r.result.success,
      );

      return {
        ...option,
        label: `${option.label} - ${numSuccess?.length} / ${allRunsMatching?.length}`,
      };
    });

    return optionsWithTestNum;
  }, [allBenchmarkRuns, latestRun]);

  const tabClass = (active: boolean) =>
    clsx(
      "px-3 py-4 text-sm border-b-2 -mb-px",
      active
        ? "border-blue-600 text-slate-900 font-medium"
        : "border-transparent text-slate-500 hover:text-slate-900",
    );

  return (
    <nav className="flex px-8 border-b border-slate-300 items-center bg-white gap-x-4">
      <div className="flex items-center gap-x-4 flex-grow">
        <div className="flex items-center gap-x-4 py-4">
          <Link to="/">
            <img src={Logo} className="w-8 h-8" />
          </Link>
          <div className="font-medium">Client Benchmark Report</div>
        </div>
        <div className="flex items-center gap-x-2 ml-4 self-stretch">
          <Link to="/" className={tabClass(!isLoadTestsRoute)}>
            Benchmarks
          </Link>
          <Link to="/load-tests/sepolia" className={tabClass(isLoadTestsRoute)}>
            Load Tests
          </Link>
        </div>
      </div>
      {!isLoadTestsRoute && !isLoading && !!allBenchmarkRuns?.runs.length && (
        <div>
          <Select
            value={benchmarkRunId}
            onChange={(e) => navigateToBenchmarkRun(e.target.value)}
          >
            {benchmarkRunOptions?.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </Select>
        </div>
      )}
      {isLoadTestsRoute &&
        !!loadTestTimestamp &&
        !isLoadingLoadTests &&
        loadTestOptions.length > 0 && (
          <div>
            <Select
              value={loadTestTimestamp}
              onChange={(e) => navigateToLoadTestRun(e.target.value)}
            >
              {loadTestOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </Select>
          </div>
        )}
    </nav>
  );
};

export default Navbar;
