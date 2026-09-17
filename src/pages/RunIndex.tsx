import { useMultipleDataSeries, useTestMetadata } from "../utils/useDataSeries";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getBenchmarkVariables } from "../filter";
import RunList from "../components/RunList";
import { BenchmarkRuns, getTestRunsWithStatus } from "../types";
import RunListFilter from "../components/RunListFilter";
import { useParams } from "react-router-dom";
import Navbar from "../components/Navbar";
import {
  gasPerSecondFromBlockMetrics,
  VALIDATOR_NEW_PAYLOAD_TIME,
} from "../utils/benchmarkThroughput";

const RunIndexInner = ({ benchmarkRuns }: { benchmarkRuns: BenchmarkRuns }) => {
  const { benchmarkRunId } = useParams();

  if (!benchmarkRunId) {
    throw new Error("Benchmark run ID is required");
  }

  const [filterSelections, setFilterSelections] = useState<
    Record<string, string | number>
  >({});

  const metricRequests = useMemo(
    () =>
      benchmarkRuns.runs.map(
        (run) =>
          [run.id, run.outputDir, "validator"] as [string, string, string],
      ),
    [benchmarkRuns.runs],
  );
  const { data: metricSeries } = useMultipleDataSeries(metricRequests);

  const testRunsWithStatus = useMemo(() => {
    if (!metricSeries) {
      return getTestRunsWithStatus(benchmarkRuns);
    }

    const runsWithMeasuredThroughput = benchmarkRuns.runs.map((run, index) => {
      if (!run.result) return run;

      const validatorGasPerSecond = gasPerSecondFromBlockMetrics(
        metricSeries[index] ?? [],
        VALIDATOR_NEW_PAYLOAD_TIME,
      );

      return {
        ...run,
        result: {
          ...run.result,
          validatorMetrics: {
            ...run.result.validatorMetrics,
            gasPerSecond: validatorGasPerSecond ?? Number.NaN,
          },
        },
      };
    });

    return getTestRunsWithStatus({ runs: runsWithMeasuredThroughput });
  }, [benchmarkRuns, metricSeries]);

  // Calculate filter options and filtered runs
  const { filterOptions, matchedRuns } = useMemo(() => {
    // Only include non-"any" filters in the params
    const activeFilters = Object.fromEntries(
      Object.entries(filterSelections).filter(([, value]) => value !== "any"),
    );

    return getBenchmarkVariables(
      testRunsWithStatus,
      {
        params: activeFilters,
        byMetric: "N/A",
      },
      undefined,
      "any",
    );
  }, [testRunsWithStatus, filterSelections]);

  // Keep all matching runs in one table. Snapshot benchmarks do not expose a
  // gas-limit dimension, so grouping by it only produces a misleading header.
  const groupedSections = useMemo(() => {
    return [
      {
        key: "all-runs",
        testName: "",
        runs: matchedRuns,
        diffKeyStart: 0,
      },
    ];
  }, [matchedRuns]);

  const autoExpand = true;

  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set(),
  );

  const groupedSectionsCached = useRef(groupedSections);
  groupedSectionsCached.current = groupedSections;
  useEffect(() => {
    if (autoExpand) {
      setExpandedSections(
        new Set(groupedSectionsCached.current.map((section) => section.key)),
      );
    } else {
      setExpandedSections(new Set());
    }
  }, [autoExpand]);

  const toggleSection = useCallback((section: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(section)) {
        next.delete(section);
      } else {
        next.add(section);
      }
      return next;
    });
  }, []);

  const updateFilterSelection = useCallback(
    (key: string, value: string | null) => {
      setFilterSelections((prev) => {
        const newSelections = { ...prev };
        if (value === null) {
          delete newSelections[key];
        } else {
          newSelections[key] = value;
        }
        return newSelections;
      });
    },
    [],
  );

  return (
    <div className="flex flex-col w-full min-h-screen">
      <Navbar />
      <div className="flex flex-col w-full flex-grow">
        <div className="overflow-x-auto p-8 pb-0 flex flex-col">
          <RunListFilter
            benchmarkRunId={benchmarkRunId}
            filterOptions={filterOptions}
            filterSelections={filterSelections}
            updateFilterSelection={updateFilterSelection}
            allRuns={testRunsWithStatus}
            testName={benchmarkRuns.runs[0]?.testName || "Benchmark"}
          />
        </div>
        <RunList
          groupedSections={groupedSections}
          expandedSections={expandedSections}
          toggleSection={toggleSection}
        />
      </div>
    </div>
  );
};

const RunIndex = () => {
  let { benchmarkRunId } = useParams();

  if (!benchmarkRunId) {
    throw new Error("Benchmark run ID is required");
  }

  const { data: allBenchmarkRuns, isLoading: isLoadingBenchmarkRuns } =
    useTestMetadata();

  const latestBenchmarkRun = useMemo(() => {
    return allBenchmarkRuns?.runs.sort(
      (a, b) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
    )[0];
  }, [allBenchmarkRuns]);

  if (latestBenchmarkRun && benchmarkRunId === "latest") {
    benchmarkRunId = `${latestBenchmarkRun.testConfig.BenchmarkRun}`;
  }

  const benchmarkRuns = useMemo((): BenchmarkRuns => {
    return {
      runs:
        allBenchmarkRuns?.runs.filter(
          (run) => run.testConfig.BenchmarkRun === benchmarkRunId,
        ) ?? [],
    };
  }, [allBenchmarkRuns, benchmarkRunId]);

  if (isLoadingBenchmarkRuns) {
    return (
      <div className="flex flex-col w-full min-h-screen">
        <Navbar />
        <div className="flex flex-col w-full flex-grow p-8 gap-4">
          <div className="animate-pulse bg-slate-200 rounded h-6 w-48" />
          <div className="animate-pulse bg-slate-100 rounded h-64 w-full" />
        </div>
      </div>
    );
  }

  return <RunIndexInner benchmarkRuns={benchmarkRuns} />;
};

export default RunIndex;
