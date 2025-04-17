import { useEffect, useMemo, useRef, useState } from "react";
import { BenchmarkRun } from "../types";
import { BenchmarkRuns, getBenchmarkVariables } from "../types";
import { isEqual } from "lodash";

export interface DataFileRequest {
  outputDir: string;
  role: string;
  name: string;
}

interface ChartSelectorProps {
  benchmarkRuns: BenchmarkRuns;
  onChangeDataQuery: (data: DataFileRequest[]) => void;
}

const camelToTitleCase = (str: string) => {
  return str
    .replace(/([A-Z])/g, " $1")
    .replace(/^./, (str) => str.toUpperCase());
};

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
    const dataToSend: DataFileRequest[] = matchedRuns.map((run) => {
      return {
        outputDir: run.outputDir,
        role: run.testConfig.role,
        name: `${run.testConfig[byMetric ?? "role"]}`,
      };
    });

    if (!isEqual(dataToSend, lastSentDataRef.current)) {
      lastSentDataRef.current = dataToSend;
      onChangeDataQuery(dataToSend);
    }
  }, [byMetric, matchedRuns, onChangeDataQuery]);

  return (
    <div className="filter-container">
      <div>
        <div>Show Line Per</div>
        <select
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
                    {val.toString()}
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
