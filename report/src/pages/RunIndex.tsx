import { Link } from "react-router-dom";
import {
  camelToTitleCase,
  formatLabel,
  formatValue,
} from "../utils/formatters";
import { useTestMetadata } from "../utils/useDataSeries";
import { useMemo, useState } from "react";
import { getBenchmarkVariables } from "../filter";

function RunIndex() {
  const { data: benchmarkRuns, isLoading: isLoadingBenchmarkRuns } =
    useTestMetadata();

  // State for filter selections
  const [filterSelections, setFilterSelections] = useState<
    Record<string, string | number>
  >({});

  // Calculate filter options and filtered runs
  const { filterOptions, matchedRuns } = useMemo(() => {
    if (!benchmarkRuns) {
      return { filterOptions: {}, matchedRuns: [] };
    }

    // Only include non-"any" filters in the params
    const activeFilters = Object.fromEntries(
      Object.entries(filterSelections).filter(([, value]) => value !== "any"),
    );

    return getBenchmarkVariables(
      benchmarkRuns.runs,
      {
        params: activeFilters,
        byMetric: "N/A",
      },
      undefined,
      "any",
    );
  }, [benchmarkRuns, filterSelections]);

  // Calculate only configs that differ across runs
  const diffConfigKeys = useMemo(() => {
    if (!matchedRuns) {
      return [];
    }

    const configKeyToValues: Record<string, Set<string | number>> = {};

    matchedRuns.forEach((run) => {
      const runConfig = run.testConfig || {};
      Object.entries(runConfig).forEach(([key, value]) => {
        if (!configKeyToValues[key]) {
          configKeyToValues[key] = new Set();
        }
        configKeyToValues[key].add(value);
      });
    });

    const differingKeys = Object.entries(configKeyToValues)
      .filter(
        ([key, values]) => values.size > 1 || filterSelections[key] !== "any",
      )
      .map(([key]) => key);

    return matchedRuns.map((run) => {
      const runConfig = run.testConfig || {};
      return Object.entries(runConfig).filter(([key]) =>
        differingKeys.includes(key),
      );
    });
  }, [matchedRuns, filterSelections]);

  if (!benchmarkRuns || isLoadingBenchmarkRuns) {
    return <div>Loading...</div>;
  }

  return (
    <div className="container mx-auto">
      {/* Filter Interface */}
      <div className="flex flex-wrap gap-4 pb-4 mb-4">
        {Object.entries(filterOptions)
          .sort((a, b) => a[0].localeCompare(b[0]))
          .map(([key, availableValues]) => {
            const currentValue = filterSelections[key] ?? "any";
            return (
              <div key={key}>
                <div className="text-sm text-slate-500 mb-1">
                  {camelToTitleCase(key)}
                </div>
                <select
                  className="bg-white border border-slate-300 rounded-md px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                  value={String(currentValue)}
                  onChange={(e) => {
                    const newValue = e.target.value;
                    setFilterSelections((prev) => {
                      const newSelections = { ...prev };
                      if (newValue === "any") {
                        delete newSelections[key];
                      } else {
                        newSelections[key] = newValue;
                      }
                      return newSelections;
                    });
                  }}
                >
                  <option value="any">Any</option>
                  {availableValues.map((val) => (
                    <option value={String(val)} key={String(val)}>
                      {formatLabel(String(val))}
                    </option>
                  ))}
                </select>
              </div>
            );
          })}
      </div>

      <table className="min-w-full divide-y divide-slate-200 rounded-lg">
        <thead>
          <tr>
            <th
              scope="col"
              className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider"
            >
              Test Name
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider"
            >
              Config
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider"
            >
              Status
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider"
            >
              Seq. Gas/s
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider"
            >
              Send Txs
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider"
            >
              Fork Choice
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider"
            >
              Get Payload
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider"
            >
              Val. Gas/s
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider"
            >
              New Payload
            </th>
            <th
              scope="col"
              className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider"
            >
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="bg-white divide-y divide-slate-200 border-left border border-slate-200">
          {matchedRuns.map((run, i) => (
            <tr key={run.outputDir} className="hover:bg-slate-50">
              <td className="px-4 py-3 whitespace-nowrap text-sm font-medium text-slate-900 align-top">
                <Link to={`/run-comparison`}>{run.testName}</Link>
              </td>
              <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-900 align-top">
                {diffConfigKeys[i] && (
                  <div className="mt-1 flex flex-wrap gap-1">
                    {[...diffConfigKeys[i]].map(([key, value]) => (
                      <span
                        key={key} // Key should be unique within a run's diff tags
                        title={`${camelToTitleCase(key)}: ${value}`}
                        className="inline-flex items-center rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-600 ring-1 ring-inset ring-slate-500/10 overflow-hidden text-ellipsis whitespace-nowrap"
                      >
                        <span className="mr-1 text-slate-500">
                          {camelToTitleCase(key)}:
                        </span>
                        {key === "GasLimit" ? (
                          <pre>{formatValue(Number(value), "gas")}</pre>
                        ) : (
                          String(formatLabel(`${value}`))
                        )}
                      </span>
                    ))}
                  </div>
                )}
              </td>
              <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                {run.result?.success ? (
                  <span className="inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700 ring-1 ring-inset ring-green-600/20">
                    Success
                  </span>
                ) : (
                  <span className="inline-flex items-center rounded-md bg-red-50 px-2 py-1 text-xs font-medium text-red-700 ring-1 ring-inset ring-red-600/10">
                    Failure
                  </span>
                )}
              </td>
              <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                {formatValue(
                  run.result?.sequencerMetrics?.gasPerSecond ?? 0,
                  "gas/s",
                )}
              </td>
              <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                {formatValue(run.result?.sequencerMetrics?.sendTxs ?? 0, "s")}
              </td>
              <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                {formatValue(
                  run.result?.sequencerMetrics?.forkChoiceUpdated ?? 0,
                  "s",
                )}
              </td>
              <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                {formatValue(
                  run.result?.sequencerMetrics?.getPayload ?? 0,
                  "s",
                )}
              </td>
              <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                {formatValue(
                  run.result?.validatorMetrics?.gasPerSecond ?? 0,
                  "gas/s",
                )}
              </td>
              <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                {formatValue(
                  run.result?.validatorMetrics?.newPayload ?? 0,
                  "s",
                )}
              </td>
              <td className="px-4 py-3 whitespace-nowrap text-sm font-medium">
                <Link
                  to={`/run-comparison`}
                  className="text-indigo-600 hover:text-indigo-900"
                >
                  Compare
                </Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default RunIndex;
