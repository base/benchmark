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
  return primaryKey;
}

const ChartGrid: React.FC<ProvidedProps> = ({ data, role }: ProvidedProps) => {
  return (
    <div className="charts-container">
      {SORTED_CHART_CONFIG.map(([metricKey, config]) => {
        const resolvedKey = resolveMetricKey(data, metricKey, config.aliases);
        const thresholdKey = role ? `${role}/${metricKey}` : null;
        const chartData = data.flatMap((s) => s.data);
        const thresholds = data[0]?.thresholds;
        const executionMetrics = chartData
          .map((d) => d.ExecutionMetrics[resolvedKey])
          .filter((v) => v !== undefined);

        if (executionMetrics.length === 0) {
          return null;
        }

        const chartProps = {
          series: data,
          metricKey: resolvedKey,
          title: config.title,
          description: config.description,
          unit: config.unit,
          thresholds,
        };

        return (
          <div key={metricKey} className="chart-container">
            <LineChart thresholdKey={thresholdKey} {...chartProps} />
          </div>
        );
      })}
    </div>
  );
};

export default ChartGrid;
