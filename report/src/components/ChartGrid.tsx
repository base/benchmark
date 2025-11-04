import React from "react";
import { SORTED_CHART_CONFIG } from "../metricDefinitions";
import { DataSeries } from "../types";
import LineChart from "./LineChart";

interface ProvidedProps {
  data: DataSeries[];
  role: "sequencer" | "validator" | null;
}

const ChartGrid: React.FC<ProvidedProps> = ({ data, role }: ProvidedProps) => {
  return (
    <div className="charts-container">
      {SORTED_CHART_CONFIG.map(([metricKey, config]) => {
        // sequencer and validator have different thresholds
        const thresholdKey = role ? `${role}/${metricKey}` : null;
        const chartData = data.flatMap((s) => s.data);
        const thresholds = data[0]?.thresholds;
        const executionMetrics = chartData
          .map((d) => d.ExecutionMetrics[metricKey])
          .filter((v) => v !== undefined);

        if (executionMetrics.length === 0) {
          return null;
        }

        const chartProps = {
          series: data,
          metricKey,
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
