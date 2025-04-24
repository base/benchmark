import { useEffect, useMemo, useRef, useState } from "react";
import { BenchmarkRun } from "../types";
import { BenchmarkRuns, getBenchmarkVariables } from "../types";
import { isEqual } from "lodash";
import {
  camelToTitleCase,
  formatValue,
  formatLabel,
} from "../utils/formatters";
import { interpolateWarm } from "d3";

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
  const [byMetric, setByMetric] = useState<string | null>("role");

  const variables = useMemo((): Record<
    string,
    (string | number | boolean)[]
  > => {
    return {
      ...getBenchmarkVariables(benchmarkRuns.runs),
      role: ["sequencer", "validator"],
    };
  }, [benchmarkRuns]);

  const [filterSelections, setFilterSelections] = useState<{
    [key: string]: string;
  }>({});

  // ensure filterSelections is a subset of variables
  useEffect(() => {
    const validVars = Object.keys(variables).filter((key) => {
      return key !== byMetric;
    });
    for (const key in filterSelections) {
      if (!validVars.includes(key)) {
        delete filterSelections[key];
      }
    }

    let newFilterSelections = filterSelections;
    for (const key of validVars) {
      if (!(key in filterSelections)) {
        newFilterSelections = {
          ...newFilterSelections,
          [key]: `${variables[key][0]}`,
        };
      }
    }

    setFilterSelections(newFilterSelections);
  }, [variables, filterSelections, byMetric]);

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
        return Object.entries(filterSelections).every(([key, value]) => {
          return (
            `${(run.testConfig as Record<string, string | number | boolean>)[key]}` ===
            `${value}`
          );
        });
      });
  }, [filterSelections, benchmarkRuns.runs]);

  const lastSentDataRef = useRef<DataFileRequest[]>([]);

  useEffect(() => {
    let colorMap: ((val: number) => string) | undefined = undefined;

    if (byMetric === "GasLimit") {
      const min = matchedRuns.reduce((a, b) => {
        return Math.min(a, Number(b.testConfig.GasLimit));
      }, 0);
      const max = matchedRuns.reduce((a, b) => {
        return Math.max(a, Number(b.testConfig.GasLimit));
      }, 0);

      console.log(min, max);

      colorMap = (val: number) =>
        interpolateWarm(1 - (max > 0 ? (val - min) / max : 0));
    }

    const dataToSend: DataFileRequest[] = matchedRuns.map((run) => {
      let seriesName = `${run.testConfig[byMetric ?? "role"]}`;
      let color = undefined;

      if (byMetric === "GasLimit") {
        seriesName = formatValue(Number(run.testConfig.GasLimit), "gas");
        color = colorMap?.(Number(run.testConfig.GasLimit));
        console.log(color);
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
  }, [byMetric, matchedRuns, onChangeDataQuery]);

  return (
    <div className="flex flex-wrap gap-4 pb-4">
      <div>
        <div>Show Line Per</div>
        <select
          className="filter-select"
          value={byMetric ?? undefined}
          onChange={(e) => setByMetric(e.target.value)}
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
        .filter(([k]) => k !== byMetric)
        .map(([key, value]) => {
          return (
            <div key={key}>
              <div>{camelToTitleCase(key)}</div>
              <select
                className="filter-select"
                value={filterSelections[key] ?? value[0]}
                onChange={(e) => {
                  setFilterSelections({
                    ...filterSelections,
                    [key]: e.target.value,
                  });
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
