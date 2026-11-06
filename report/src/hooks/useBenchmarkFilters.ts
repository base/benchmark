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

  // Group-by dimension to actually use: the selected/default metric only works
  // if it is a real multi-value variable. When it is not present (e.g. the
  // "role" default collapses to a single value for sequencer-only runs), fall
  // back to the first available variable so grouping still works — this is what
  // lets you compare by Transaction Payload when there is only one role.
  const effectiveByMetric = useMemo(() => {
    const requested = filterSelections.byMetric || defaultMetric;
    if (variables[requested]) {
      return requested;
    }
    const available = Object.keys(variables);
    return available.length > 0 ? available[0] : requested;
  }, [filterSelections.byMetric, defaultMetric, variables]);

  // Calculate current options and matched runs based on current selections + variables
  const { filterOptions, matchedRuns } = useMemo(() => {
    const currentSelections = {
      ...filterSelections,
      byMetric: effectiveByMetric,
    };
    // Pass memoized variables to avoid recalculating them inside
    return getBenchmarkVariables(
      runsWithRoles,
      currentSelections,
      variables,
      "first",
    );
  }, [runsWithRoles, filterSelections, effectiveByMetric, variables]);

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

  // get the role if not grouped by role
  const role = useMemo(() => {
    if (effectiveByMetric === "role") {
      return null;
    }

    // variables.role is absent when every run shares a single role (a
    // one-value dimension is dropped from `variables`), so fall back to the
    // role carried on the runs themselves, then to "sequencer". Reading
    // variables.role[0] directly would throw when only one role exists and
    // break every non-role group-by (e.g. line-per-TransactionPayload).
    const roleFromData = runsWithRoles.find((r) => r.testConfig.role)
      ?.testConfig.role as "sequencer" | "validator" | undefined;

    return (
      (filterSelections.params.role as "sequencer" | "validator") ??
      variables.role?.[0] ??
      roleFromData ??
      "sequencer"
    );
  }, [
    effectiveByMetric,
    filterSelections.params.role,
    variables,
    runsWithRoles,
  ]);

  return {
    variables,
    filterOptions,
    matchedRuns,
    filterSelections, // Return current selections for UI binding
    byMetric: effectiveByMetric, // Effective group-by (falls back when default is unavailable)
    setFilters, // Return the simplified setter
    setByMetric,
    role,
  };
}
