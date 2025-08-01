import React from "react";
import { CHART_CONFIG } from "../metricDefinitions";
import { DataSeries } from "../types";
import LineChart from "./LineChart";

interface ProvidedProps {
  data: DataSeries[];
}

const ChartGrid: React.FC<ProvidedProps> = ({ data }: ProvidedProps) => {
  return (
    <div className="charts-container">
      {Object.entries(CHART_CONFIG).map(([metricKey, config]) => {
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
            <LineChart {...chartProps} />
          </div>
        );
      })}
    </div>
  );
};

export default ChartGrid;
