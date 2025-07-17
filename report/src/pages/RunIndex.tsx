import { useTestMetadata } from "../utils/useDataSeries";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getBenchmarkVariables } from "../filter";
import RunList from "../components/RunList";
import { BenchmarkRuns, getTestRunsWithStatus } from "../types";
import { groupBy } from "lodash";
import RunListFilter from "../components/RunListFilter";
import { useParams } from "react-router-dom";
import Navbar from "../components/Navbar";

const RunIndexInner = ({ benchmarkRuns }: { benchmarkRuns: BenchmarkRuns }) => {
  const { benchmarkRunId } = useParams();

  if (!benchmarkRunId) {
    throw new Error("Benchmark run ID is required");
  }

  const [filterSelections, setFilterSelections] = useState<
    Record<string, string | number>
  >({});

  const testRunsWithStatus = useMemo(
    () => getTestRunsWithStatus(benchmarkRuns),
    [benchmarkRuns],
  );

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

  // Group matchedRuns by id and precompute group sections with diffKeyStart
  const groupedSections = useMemo(() => {
    // Group runs by id
    matchedRuns.forEach((run) => {
      if (!run.id) {
        run.id = run.outputDir;
      }
    });

    const groups = groupBy(matchedRuns, "testConfig.GasLimit");

    // Build sections array with diffKeyStart
    const sections: {
      key: string;
      testName: string;
      runs: typeof matchedRuns;
      diffKeyStart: number;
    }[] = [];
    let diffKeyStart = 0;
    Object.entries(groups).forEach(([id, runs]) => {
      sections.push({
        key: id,
        testName: runs[0]?.testName || id,
        runs,
        diffKeyStart,
      });
      diffKeyStart += runs.length;
    });
    return sections;
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

  if (!benchmarkRuns || isLoadingBenchmarkRuns) {
    return <div>Loading...</div>;
  }

  return <RunIndexInner benchmarkRuns={benchmarkRuns} />;
};

export default RunIndex;
