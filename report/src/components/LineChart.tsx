import React from "react";
import * as d3 from "d3";
import { DataSeries, MetricData, ChartConfig } from "../types";
import BaseChart from "./BaseChart";
import { formatValue } from "../utils/formatters";

interface LineChartProps {
  series: DataSeries[];
  metricKey: string;
  title?: string;
  description?: string;
  unit?: ChartConfig["unit"];
}

const LineChart: React.FC<LineChartProps> = ({
  series,
  metricKey,
  title,
  description,
  unit,
}) => {
  // Get all data points for domain calculation
  const allData = series.flatMap((s) => s.data);

  return (
    <BaseChart
      data={allData}
      metricKey={metricKey}
      title={title}
      description={description}
    >
      {(svg, dimensions) => {
        // Create an array of all block numbers
        const blockNumbers = allData.map((d) => d.BlockNumber);
        const minBlock = Math.min(...blockNumbers);
        const maxBlock = Math.max(...blockNumbers);

        const x = d3
          .scaleLinear()
          .domain([minBlock, maxBlock])
          .range([0, dimensions.width]);

        const y = d3
          .scaleLinear()
          .domain([
            0,
            d3.max(allData, (d) => d.ExecutionMetrics[metricKey]) as number,
          ])
          .range([dimensions.height, 0]);

        // Add grid lines
        svg
          .append("g")
          .attr("class", "grid")
          .attr("transform", `translate(0,${dimensions.height})`)
          .call(
            d3
              .axisBottom(x)
              .tickSize(-dimensions.height)
              .tickFormat(() => ""),
          )
          .style("stroke-dasharray", "3,3")
          .style("stroke-opacity", 0.2);

        svg
          .append("g")
          .attr("class", "grid")
          .call(
            d3
              .axisLeft(y)
              .tickSize(-dimensions.width)
              .tickFormat(() => ""),
          )
          .style("stroke-dasharray", "3,3")
          .style("stroke-opacity", 0.2);

        // Add axes
        svg
          .append("g")
          .attr("transform", `translate(0,${dimensions.height})`)
          .call(d3.axisBottom(x))
          .selectAll("text")
          .style("text-anchor", "end")
          .attr("dx", "-.8em")
          .attr("dy", ".15em")
          .attr("transform", "rotate(-45)");

        svg
          .append("g")
          .call(
            d3
              .axisLeft(y)
              .tickFormat((d) => formatValue(d as number, unit))
              .ticks(8),
          )
          .append("text")
          .attr("fill", "#000")
          .attr("transform", "rotate(-90)")
          .attr("y", 6)
          .attr("dy", ".71em")
          .style("text-anchor", "end");

        // Add legend
        const legend = svg
          .append("g")
          .attr("class", "legend")
          .attr("font-family", "sans-serif")
          .attr("font-size", 10)
          .attr("text-anchor", "start")
          .selectAll("g")
          .data(series)
          .join("g")
          .attr(
            "transform",
            (_d, i) => `translate(${i * 100}, ${dimensions.height + 40})`,
          ); // Position below chart

        legend
          .append("rect")
          .attr("x", 0)
          .attr("width", 10) // Smaller square
          .attr("height", 10) // Smaller square
          .attr("fill", (d, i) => d.color || d3.schemeCategory10[i % 10]);

        legend
          .append("text")
          .attr("x", 15) // Adjust text position
          .attr("y", 5) // Adjust text position (center vertically)
          .attr("dy", "0.35em")
          .text((d) => d.name);

        // Center the legend group horizontally
        const legendGroupSelection = svg.selectAll(
          ".legend > g",
        ) as d3.Selection<SVGGElement, DataSeries, SVGGElement, unknown>;
        let totalLegendWidth = 0;
        const legendItemWidths: number[] = [];

        legendGroupSelection.each(function () {
          const bbox = (this as SVGGraphicsElement).getBBox();
          legendItemWidths.push(bbox.width);
          totalLegendWidth += bbox.width;
        });

        const spacing = 10;
        totalLegendWidth += Math.max(0, series.length - 1) * spacing; // Add spacing between items

        const startX = Math.max(0, (dimensions.width - totalLegendWidth) / 2);
        let currentX = startX;

        legendGroupSelection.each(function (_d, i) {
          d3.select(this).attr(
            "transform",
            `translate(${currentX}, ${dimensions.height + 25})`,
          ); // Move legend closer
          currentX += legendItemWidths[i] + spacing; // Add spacing between items
        });

        series.forEach((s, i) => {
          const color = s.color || d3.schemeCategory10[i % 10];

          // Add the line
          const line = d3
            .line<MetricData>()
            .x((d) => x(d.BlockNumber))
            .y((d) => y(d.ExecutionMetrics[metricKey]));

          svg
            .append("path")
            .datum(s.data)
            .attr("fill", "none")
            .attr("stroke", color)
            .attr("stroke-width", 2)
            .attr("d", line);

          // Add dots
          svg
            .selectAll(`.dot-${i}`)
            .data(s.data)
            .enter()
            .append("circle")
            .attr("class", `dot-${i}`)
            .attr("cx", (d) => x(d.BlockNumber))
            .attr("cy", (d) => y(d.ExecutionMetrics[metricKey]))
            .attr("r", 4)
            .style("fill", color)
            .style("opacity", 0);
        });
      }}
    </BaseChart>
  );
};

export default LineChart;
