import React, { useEffect, useMemo, useRef, useState } from "react";
import { DataSeries } from "./types";
import { BenchmarkRuns, getBenchmarkVariables, MetricData } from "./types";
import LineChart from "./components/LineChart";
import BarChart from "./components/BarChart";
import { CHART_CONFIG } from "./chart-manifest";

interface ChartSelectorProps {
  benchmarkRuns: BenchmarkRuns;

  fetchMetrics: (outputDir: string, role: string) => Promise<MetricData[]>;
  fetchResult: (outputDir: string, role: string) => Promise<unknown>;
  getLogsDownloadGz: (outputDir: string, role: string) => string;
}

const ChartSelector = ({
  benchmarkRuns,
  fetchMetrics,
  fetchResult,
  getLogsDownloadGz,
}: ChartSelectorProps) => {
  const [data, setData] = useState<DataSeries[]>([]);

  const [byMetric, setByMetric] = useState<string | null>("role");

  const variables = useMemo((): Record<string, (string | number | boolean)[]> => {
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

    console.log(validVars)

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

  console.log(filterSelections)

  const matchedRuns = useMemo(() => {
    return benchmarkRuns.runs.flatMap((r) => ([{
      ...r,
      testConfig: {
        ...r.testConfig,
        role: 'sequencer'
      },
    }, {
      ...r,
      testConfig: {
        ...r.testConfig,
        role: 'validator'
      },
    }])).filter((run) => {
      return Object.entries(filterSelections).every(([key, value]) => {
        return `${(run.testConfig as Record<string, string | number | boolean>)[key]}` === `${value}`;
      });
    });
  }, [filterSelections, benchmarkRuns.runs]);

  const urlsToFetch = useMemo(() => {
    const urls = new Set(
      matchedRuns.map((run) => `${run.outputDir},${run.testConfig.role},${run.testConfig[byMetric ?? "role"]}`),
    );

    return [...urls];
  }, [matchedRuns]);

  console.log(urlsToFetch)

  const loadingRef = useRef<number>(0);
  useEffect(() => {
    loadingRef.current += 1;
    const mustEqual = loadingRef.current;
    setData([]);

    (async () => {
      
        const result = (
          await Promise.all(
            urlsToFetch.map(async (url) => {
              const [outputDir, role, name] = url.split(",");
              const data = (await fetchMetrics(outputDir, role))
              
              return { data, name: `${name}`  };
            }),
          )
        ).flat()

        if (loadingRef.current === mustEqual) {
          setData(result);
        }

    })();
  }, [urlsToFetch]);

  return (
    <>
      <div className="filter-container">
        <div>
          <div>Show Line Per</div>
          <select
            value={byMetric ?? undefined}
            onChange={(e) => setByMetric(e.target.value)}
          >
            {Object.entries(variables).map(([k]) => (
              <option value={`${k}`}>{k}</option>
            ))}
          </select>
        </div>
        {Object.entries(variables)
          .sort((a, b) => a[0].localeCompare(b[0]))
          .filter(([k]) => k !== byMetric)
          .map(([key, value]) => {
            return (
              <div key={key}>
                <div>{key}</div>
                <select value={filterSelections[key] ?? value[0]}
                  onChange={(e) => {
                    setFilterSelections({
                      ...filterSelections,
                      [key]: e.target.value,
                    });
                  }}
                >
                  {value.map((val, i) => (
                    <option value={`${val}`}>{val.toString()}</option>
                  ))}
                </select>
              </div>
            );
          })}
      </div>
      <div className="charts-container">
        {Object.entries(CHART_CONFIG).map(([metricKey, config]) => {
          const chartProps = {
            series: data,
            metricKey,
            title: config.title,
            description: config.description,
            unit: config.unit,
          };

          return (
            <div key={metricKey} className="chart-container">
              {config.type === "line" ? (
                <LineChart {...chartProps} />
              ) : (
                <BarChart {...chartProps} />
              )}
            </div>
          );
        })}
      </div>
    </>
  );
};

export default ChartSelector;
