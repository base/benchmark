// Export this type for use elsewhere
export type FilterValue = string | number | boolean;
type FilterSelectionsParams = Record<string, FilterValue>;

/**
 * Matches runs against a given set of filter criteria.
 */
function matchRuns<T extends { testConfig: Record<string, string | number> }>(
  runs: T[],
  filterSelections: FilterSelectionsParams,
): T[] {
  return runs.filter((run) => {
    return Object.entries(filterSelections).every(([key, value]) => {
      return `${run.testConfig[key]}` === `${value}`;
    });
  });
}

/**
 * Extracts variables, calculates available filter options, and filters runs based on selections.
 * Ensures that filter options remain available even if the current selection yields no results.
 */
export function getBenchmarkVariables<
  T extends { testConfig: Record<string, string | number> },
>(
  runs: T[],
  filterSelections: {
    params: FilterSelectionsParams;
    byMetric: string;
  },
  // Add optional argument for pre-calculated variables
  precalculatedVariables?: Record<string, FilterValue[]> | undefined,
  defaultBehavior: "first" | "any" = "first", // if a filter is not set, should it be any or first
) {
  const variables =
    precalculatedVariables ??
    (() => {
      const allPossibleValues: Record<string, Set<FilterValue>> = {};
      for (const run of runs) {
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
    })(); // Immediately invoke the IIFE if needed

  const filterOptions: Record<string, FilterValue[]> = {};
  for (const key of Object.keys(variables)) {
    // Skip the metric used for grouping series
    if (key === filterSelections.byMetric) {
      continue;
    }

    // Determine the base filters to use when checking options for `key`
    // This includes all *other* active filters.
    const otherActiveFilters = { ...filterSelections.params };
    delete otherActiveFilters[key];
    delete otherActiveFilters[filterSelections.byMetric];

    // Check which values for `key` yield results when combined with `otherActiveFilters`
    const validValuesForKey = variables[key].filter((value) => {
      const potentialFilters = { ...otherActiveFilters, [key]: value };
      return matchRuns(runs, potentialFilters).length > 0;
    });

    if (validValuesForKey.length > 0) {
      filterOptions[key] = validValuesForKey;
    }
  }

  // 3. Calculate active filters with defaults
  const activeFilters: FilterSelectionsParams = {};
  for (const [key, values] of Object.entries(filterOptions)) {
    if (key === filterSelections.byMetric) continue;

    const selectedValue = filterSelections.params[key];
    if (selectedValue !== undefined) {
      activeFilters[key] = selectedValue;
    } else if (defaultBehavior === "first" && values.length > 0) {
      activeFilters[key] = values[0];
    }
  }

  // 4. Calculate the final matched runs based on the current activeFilters
  const matchedRuns = matchRuns(runs, activeFilters);

  return { variables, filterOptions, matchedRuns };
}
