import { useCallback, useMemo } from "react";
import { isEqual } from "lodash";
import { type BenchmarkRun } from "../types";
import { useSearchParamsState } from "../utils/useSearchParamsState";
import { getBenchmarkVariables } from "../filter";
type FilterValue = string | number | boolean;
type FilterSelectionsParams = Record<string, FilterValue>;
type FilterSelections = {
  params: FilterSelectionsParams;
  byMetric: string;
};

/**
 * Custom hook to manage benchmark filter selections and derived data.
 * Encapsulates state logic and ensures filter consistency after updates.
 *
 * @param benchmarkRuns - The raw benchmark runs data.
 * @param defaultMetric - The default metric to group by ('role' if not specified).
 * @returns An object containing derived data and state management functions.
 */
export function useBenchmarkFilters(
  runsWithRoles: BenchmarkRun[],
  defaultMetric: string = "role",
) {
  const [filterSelections, setRawFilterSelections] =
    useSearchParamsState<FilterSelections>("filters", {
      params: {},
      byMetric: defaultMetric,
    });

  // Memoize variables once, as they don't depend on selections
  const variables = useMemo(() => {
    const allPossibleValues: Record<string, Set<FilterValue>> = {};
    for (const run of runsWithRoles) {
      for (const [key, value] of Object.entries(run.testConfig)) {
        if (!allPossibleValues[key]) {
          allPossibleValues[key] = new Set();
        }
        allPossibleValues[key].add(value);
      }
    }
    return Object.fromEntries(
      Object.entries(allPossibleValues)
        .filter(([, values]) => values.size > 1)
        .map(([key, values]) => [key, [...values].sort()]),
    );
  }, [runsWithRoles]);

  // Calculate current options and matched runs based on current selections + variables
  const { filterOptions, matchedRuns } = useMemo(() => {
    // Ensure a default byMetric if somehow cleared
    const currentSelections = {
      ...filterSelections,
      byMetric: filterSelections.byMetric || defaultMetric,
    };
    // Pass memoized variables to avoid recalculating them inside
    return getBenchmarkVariables(
      runsWithRoles,
      currentSelections,
      variables,
      "first",
    );
  }, [runsWithRoles, filterSelections, defaultMetric, variables]);

  // Define the setter function (simplified: no adjustment logic)
  const setFilters = useCallback(
    (name: string, value: FilterValue) => {
      const prevState = filterSelections;

      const targetParams = {
        ...prevState.params,
        [name]: value,
      };

      const targetFilterSelections = {
        ...prevState,
        params: targetParams,
      };

      if (!isEqual(targetFilterSelections, prevState)) {
        setRawFilterSelections(targetFilterSelections);
      }
    },
    [filterSelections, setRawFilterSelections],
  );

  const setByMetric = useCallback(
    (metric: string) => {
      // when by metric changes, reset all other filters

      setRawFilterSelections({
        params: {},
        byMetric: metric,
      });
    },
    [setRawFilterSelections],
  );

  return {
    variables,
    filterOptions,
    matchedRuns,
    filterSelections, // Return current selections for UI binding
    setFilters, // Return the simplified setter
    setByMetric,
  };
}
