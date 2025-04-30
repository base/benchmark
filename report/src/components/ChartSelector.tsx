import { useEffect, useMemo, useRef } from "react";
import { BenchmarkRuns } from "../types";
import { isEqual } from "lodash";
import {
  camelToTitleCase,
  formatValue,
  formatLabel,
} from "../utils/formatters";
import { interpolateWarm } from "d3";
import { useBenchmarkFilters } from "../hooks/useBenchmarkFilters";

export interface DataFileRequest {
  outputDir: string;
  role: string;
  name: string;
  color?: string;
}

interface ChartSelectorProps {
  benchmarkRuns: BenchmarkRuns;
  onChangeDataQuery: (data: DataFileRequest[]) => void;
}

const ChartSelector = ({
  benchmarkRuns,
  onChangeDataQuery,
}: ChartSelectorProps) => {
  const runsWithRoles = useMemo(
    () =>
      benchmarkRuns.runs.flatMap((r) => [
        { ...r, testConfig: { ...r.testConfig, role: "sequencer" } },
        { ...r, testConfig: { ...r.testConfig, role: "validator" } },
      ]),
    [benchmarkRuns.runs],
  );

  const {
    variables,
    filterOptions,
    matchedRuns,
    filterSelections,
    setFilters,
    setByMetric,
  } = useBenchmarkFilters(runsWithRoles, "role");

  const lastSentDataRef = useRef<DataFileRequest[]>([]);

  useEffect(() => {
    let colorMap: ((val: number) => string) | undefined = undefined;

    if (filterSelections.byMetric === "GasLimit" && matchedRuns.length > 0) {
      const gasLimits = matchedRuns.map((r) => Number(r.testConfig.GasLimit));
      const min = Math.min(...gasLimits);
      const max = Math.max(...gasLimits);

      colorMap = (val: number) =>
        interpolateWarm(max - min > 0 ? 1 - (val - min) / (max - min) : 0.5);
    }

    const dataToSend: DataFileRequest[] = matchedRuns
      .map((run): DataFileRequest | null => {
        if (!run.testConfig || !run.outputDir) {
          console.warn("Skipping run with missing data:", run);
          return null;
        }

        let seriesName: string;
        let color: string | undefined = undefined;
        const byMetricValue = run.testConfig[filterSelections.byMetric];

        if (filterSelections.byMetric === "GasLimit") {
          const gasLimitNum = Number(byMetricValue);
          seriesName = formatValue(gasLimitNum, "gas");
          color = colorMap?.(gasLimitNum);
        } else {
          seriesName =
            byMetricValue !== undefined
              ? formatLabel(String(byMetricValue))
              : "Unknown";
        }

        const role = run.testConfig.role ?? "unknown";

        const request: DataFileRequest = {
          outputDir: run.outputDir,
          role: String(role),
          name: seriesName,
        };
        if (color !== undefined) {
          request.color = color;
        }
        return request;
      })
      .filter((item): item is DataFileRequest => item !== null);

    if (!isEqual(dataToSend, lastSentDataRef.current)) {
      lastSentDataRef.current = dataToSend;
      onChangeDataQuery(dataToSend);
    }
  }, [matchedRuns, filterSelections.byMetric, onChangeDataQuery]);

  return (
    <div className="flex flex-wrap gap-4 pb-4">
      <div>
        <div>Show Line Per</div>
        <select
          className="filter-select"
          value={filterSelections.byMetric}
          onChange={(e) => setByMetric(e.target.value)}
        >
          {Object.keys(variables).map((k) => (
            <option value={k} key={k}>
              {camelToTitleCase(k)}
            </option>
          ))}
        </select>
      </div>
      {Object.entries(filterOptions)
        .sort((a, b) => a[0].localeCompare(b[0]))
        .filter(([k]) => k !== filterSelections.byMetric)
        .map(([key, availableValues]) => {
          const currentValue =
            filterSelections.params[key] ?? availableValues[0];
          return (
            <div key={key}>
              <div>{camelToTitleCase(key)}</div>
              <select
                className="filter-select"
                value={String(currentValue)}
                onChange={(e) => {
                  setFilters(key, e.target.value);
                }}
              >
                {availableValues.map((val) => (
                  <option value={String(val)} key={String(val)}>
                    {formatLabel(String(val))}
                  </option>
                ))}
              </select>
            </div>
          );
        })}
      {matchedRuns.length === 0 && (
        <div className="w-full text-center text-gray-500 italic py-2">
          No benchmark runs match the current filter combination.
        </div>
      )}
    </div>
  );
};

export default ChartSelector;
