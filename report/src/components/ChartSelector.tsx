import { useEffect, useMemo, useRef } from "react";
import { BenchmarkRun } from "../types";
import { BenchmarkRuns, getBenchmarkVariables } from "../types";
import { isEqual } from "lodash";
import {
  camelToTitleCase,
  formatValue,
  formatLabel,
} from "../utils/formatters";
import { interpolateWarm } from "d3";
import { useSearchParamsState } from "../utils/useSearchParamsState";

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

interface BenchmarkRunWithRole extends BenchmarkRun {
  testConfig: BenchmarkRun["testConfig"] & {
    role: string;
  };
}

const ChartSelector = ({
  benchmarkRuns,
  onChangeDataQuery,
}: ChartSelectorProps) => {
  const variables = useMemo((): Record<
    string,
    (string | number | boolean)[]
  > => {
    return {
      ...getBenchmarkVariables(benchmarkRuns.runs),
      role: ["sequencer", "validator"],
    };
  }, [benchmarkRuns]);

  const [filterSelections, setFilterSelections] = useSearchParamsState<{
    params: { [key: string]: string };
    byMetric: string;
  }>("filters", { params: {}, byMetric: "role" });

  const validFilterSelections = useMemo(() => {
    return Object.fromEntries(
      Object.keys(variables)
        .filter((key) => {
          return key !== filterSelections.byMetric;
        })
        .map(
          (key) =>
            [key, filterSelections.params[key] ?? variables[key][0]] as const,
        ),
    );
  }, [filterSelections.params, filterSelections.byMetric]);

  const matchedRuns = useMemo(() => {
    return benchmarkRuns.runs
      .flatMap((r): BenchmarkRunWithRole[] => [
        {
          ...r,
          testConfig: {
            ...r.testConfig,
            role: "sequencer",
          },
        },
        {
          ...r,
          testConfig: {
            ...r.testConfig,
            role: "validator",
          },
        },
      ])
      .filter((run) => {
        return Object.entries(validFilterSelections).every(([key, value]) => {
          return (
            `${(run.testConfig as Record<string, string | number | boolean>)[key]}` ===
            `${value}`
          );
        });
      });
  }, [validFilterSelections, benchmarkRuns.runs]);

  const lastSentDataRef = useRef<DataFileRequest[]>([]);

  useEffect(() => {
    let colorMap: ((val: number) => string) | undefined = undefined;

    if (filterSelections.byMetric === "GasLimit") {
      const min = matchedRuns.reduce((a, b) => {
        return Math.min(a, Number(b.testConfig.GasLimit));
      }, 0);
      const max = matchedRuns.reduce((a, b) => {
        return Math.max(a, Number(b.testConfig.GasLimit));
      }, 0);

      colorMap = (val: number) =>
        interpolateWarm(1 - (max > 0 ? (val - min) / max : 0));
    }

    const dataToSend: DataFileRequest[] = matchedRuns.map((run) => {
      let seriesName = `${run.testConfig[filterSelections.byMetric ?? "role"]}`;
      let color = undefined;

      if (filterSelections.byMetric === "GasLimit") {
        seriesName = formatValue(Number(run.testConfig.GasLimit), "gas");
        color = colorMap?.(Number(run.testConfig.GasLimit));
      }

      return {
        outputDir: run.outputDir,
        role: run.testConfig.role,
        name: seriesName,
        color,
      };
    });

    if (!isEqual(dataToSend, lastSentDataRef.current)) {
      lastSentDataRef.current = dataToSend;
      onChangeDataQuery(dataToSend);
    }
  }, [filterSelections, matchedRuns, onChangeDataQuery]);

  return (
    <div className="flex flex-wrap gap-4 pb-4">
      <div>
        <div>Show Line Per</div>
        <select
          className="filter-select"
          value={filterSelections.byMetric ?? undefined}
          onChange={(e) =>
            setFilterSelections((fs) => ({
              ...fs,
              byMetric: e.target.value,
            }))
          }
        >
          {Object.entries(variables).map(([k]) => (
            <option value={`${k}`} key={k}>
              {camelToTitleCase(k)}
            </option>
          ))}
        </select>
      </div>
      {Object.entries(variables)
        .sort((a, b) => a[0].localeCompare(b[0]))
        .filter(([k]) => k !== filterSelections.byMetric)
        .map(([key, value]) => {
          return (
            <div key={key}>
              <div>{camelToTitleCase(key)}</div>
              <select
                className="filter-select"
                value={filterSelections.params[key] ?? value[0]}
                onChange={(e) => {
                  setFilterSelections((fs) => ({
                    ...fs,
                    params: { ...fs.params, [key]: e.target.value },
                  }));
                }}
              >
                {value.map((val) => (
                  <option value={`${val}`} key={`${val}`}>
                    {formatLabel(val.toString())}
                  </option>
                ))}
              </select>
            </div>
          );
        })}
    </div>
  );
};

export default ChartSelector;
