import React from "react";
import { SORTED_CHART_CONFIG } from "../metricDefinitions";
import { DataSeries } from "../types";
import LineChart from "./LineChart";

interface ProvidedProps {
  data: DataSeries[];
  role: "sequencer" | "validator" | null;
}

function resolveMetricKey(
  data: DataSeries[],
  primaryKey: string,
  aliases: string[] = [],
): string {
  const keys = [primaryKey, ...aliases];
  const chartData = data.flatMap((s) => s.data);
  for (const key of keys) {
    if (chartData.some((d) => d.ExecutionMetrics[key] !== undefined)) {
      return key;
    }
  }

  const metricKeys = chartData.flatMap((d) => Object.keys(d.ExecutionMetrics));
  for (const key of keys) {
    const quantileSuffix = key.match(/(_quantile_\d+(?:_\d+)?)$/)?.[1] ?? "";
    const metricPrefix = quantileSuffix
      ? key.slice(0, -quantileSuffix.length)
      : key;
    const labeledMetricKeys = metricKeys.filter(
      (metricKey) =>
        metricKey.startsWith(`${metricPrefix}_`) &&
        (!quantileSuffix || metricKey.endsWith(quantileSuffix)),
    );
    if (labeledMetricKeys.length === 1) {
      return labeledMetricKeys[0];
    }
  }
  return primaryKey;
}

const GROUP_ORDER = ["Latency", "Chain", "Throughput"];

const ChartGrid: React.FC<ProvidedProps> = ({ data, role }: ProvidedProps) => {
  const chartData = data.flatMap((s) => s.data);
  const thresholds = data[0]?.thresholds;

  const visibleCharts = SORTED_CHART_CONFIG.flatMap(([metricKey, config]) => {
    const resolvedKey = resolveMetricKey(data, metricKey, config.aliases);
    const executionMetrics = chartData
      .map((d) => d.ExecutionMetrics[resolvedKey])
      .filter((v) => v !== undefined);
    if (executionMetrics.length === 0) return [];
    return [{ metricKey, resolvedKey, config }];
  });

  const grouped = visibleCharts.reduce<Record<string, typeof visibleCharts>>(
    (acc, item) => {
      const group = item.config.group ?? "Other";
      if (!acc[group]) acc[group] = [];
      acc[group].push(item);
      return acc;
    },
    {},
  );

  const groupKeys = [
    ...GROUP_ORDER.filter((g) => grouped[g]),
    ...Object.keys(grouped).filter((g) => !GROUP_ORDER.includes(g)),
  ];

  return (
    <div className="charts-container">
      {groupKeys.map((group) => (
        <div key={group} className="metric-group">
          <h2 className="metric-group-title">{group}</h2>
          <div className="metric-group-charts">
            {grouped[group].map(({ metricKey, resolvedKey, config }) => {
              const thresholdKey = role ? `${role}/${metricKey}` : null;
              return (
                <div key={metricKey} className="chart-container">
                  <LineChart
                    thresholdKey={thresholdKey}
                    series={data}
                    metricKey={resolvedKey}
                    title={config.title}
                    description={config.description}
                    unit={config.unit}
                    thresholds={thresholds}
                  />
                </div>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
};

export default ChartGrid;
