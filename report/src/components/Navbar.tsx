import {
  Link,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";
import Logo from "../assets/logo.svg";
import { useTestMetadata } from "../utils/useDataSeries";
import { useCallback, useMemo } from "react";
import { uniqBy } from "lodash";
import {} from "react-router-dom";
import Select from "./Select";

interface ProvidedProps {
  urlPrefix?: string;
}

const Navbar = ({ urlPrefix }: ProvidedProps) => {
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

  const { benchmarkRunId } = useParams();

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

  return (
    <nav className="flex px-8 border-b border-slate-300 items-center bg-white gap-x-4">
      <div className="flex items-center gap-x-4 flex-grow">
        <div className="flex items-center gap-x-4 py-4">
          <Link to="/">
            <img src={Logo} className="w-8 h-8" />
          </Link>
          <div className="font-medium">Client Benchmark Report</div>
        </div>
      </div>
      {!isLoading && !!allBenchmarkRuns?.runs.length && (
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
    </nav>
  );
};

export default Navbar;
