import React, { useRef, useEffect, useCallback, useId, useState } from "react";
import * as d3 from "d3";
import { DataSeries, MetricData, ChartConfig } from "../types";
import BaseChart from "./BaseChart";
import { formatValue, formatLabel } from "../utils/formatters";
import calculateTooltipLayout from "../hooks/useTooltipLayout";
import useChartHoverSync, { ChartHoverEvent } from "../hooks/useChartHoverSync";
import { isEqual } from "lodash";

interface LineChartProps {
  series: DataSeries[];
  metricKey: string;
  title?: string;
  description?: string;
  unit?: ChartConfig["unit"];
  xAxisDomain?: [number, number];
  xAxisLabel?: string;
}

interface TooltipData {
  id: number;
  x: number;
  y: number;
  width: number;
  height: number;
  value: number;
  originalY: number;
  seriesName: string;
  formattedValue: string;
  color: string;
}

const LineChart: React.FC<LineChartProps> = ({
  series,
  metricKey,
  title,
  description,
  unit,
  xAxisDomain,
  xAxisLabel = "Block Number",
}) => {
  // Generate a unique ID for this chart
  const chartId = useId();

  // Get all data points for domain calculation
  const allData = series.flatMap((s) => s.data);

  // State to track current tooltips
  // Only tooltips are stored in state to trigger rerenders
  const [tooltips, setTooltips] = useState<TooltipData[]>([]);

  // Use ref for tracking mouse position to avoid rerenders
  const mouseXRef = useRef<number | null>(null);

  // Refs for chart elements
  const chartRef = useRef<{ height: number; width: number }>({
    height: 0,
    width: 0,
  });
  const svgRef = useRef<SVGGElement | null>(null);
  const xScaleRef = useRef<d3.ScaleLinear<number, number> | null>(null);
  const yScaleRef = useRef<d3.ScaleLinear<number, number> | null>(null);
  const hoverGroupRef = useRef<d3.Selection<
    SVGGElement,
    unknown,
    null,
    undefined
  > | null>(null);
  const tooltipContainerRef = useRef<d3.Selection<
    SVGGElement,
    unknown,
    null,
    undefined
  > | null>(null);
  const verticalLineRef = useRef<d3.Selection<
    SVGLineElement,
    unknown,
    null,
    undefined
  > | null>(null);

  // Filter out any data points that have undefined or NaN values for this metric
  const validData = allData.filter((d) => {
    const val = d.ExecutionMetrics[metricKey];
    return val !== undefined && !isNaN(val);
  });

  // Calculate adjusted tooltip positions using the pure function
  const adjustedTooltips = calculateTooltipLayout(
    tooltips,
    chartRef.current?.height || 0,
  );

  // Create scales outside the render function to avoid recreation
  const blockNumbers = validData.map((d) => d.BlockNumber);
  const minBlock = xAxisDomain
    ? xAxisDomain[0]
    : blockNumbers.length
      ? Math.min(...blockNumbers)
      : 0;
  const maxBlock = xAxisDomain
    ? xAxisDomain[1]
    : blockNumbers.length
      ? Math.max(...blockNumbers)
      : 100;

  // Find the max value, with fallback to avoid NaN
  const maxValue =
    (d3.max(validData, (d) => d.ExecutionMetrics[metricKey]) as number) || 0;

  // Function to calculate tooltips based on mouse position
  const calculateTooltips = useCallback(
    (mouseX: number): TooltipData[] => {
      if (
        !chartRef.current ||
        !xScaleRef.current ||
        !yScaleRef.current ||
        !svgRef.current
      ) {
        return [];
      }

      const x = xScaleRef.current;
      const y = yScaleRef.current;

      // Ensure mouseX is within bounds
      const boundedX = Math.max(0, Math.min(chartRef.current.width, mouseX));

      // Convert back to data space
      const xValue = x.invert(boundedX);

      // Calculate tooltip positions
      const newTooltips: TooltipData[] = [];

      series.forEach((s, i) => {
        // Skip if series has no data
        if (!s.data.length) return;

        // Find the closest point in the series data
        const bisect = d3.bisector((d: MetricData) => d.BlockNumber).left;
        const index = bisect(s.data, xValue);

        // Handle edge cases
        const point =
          index >= s.data.length
            ? s.data[s.data.length - 1]
            : index <= 0
              ? s.data[0]
              : Math.abs(s.data[index].BlockNumber - xValue) <
                  Math.abs(s.data[index - 1].BlockNumber - xValue)
                ? s.data[index]
                : s.data[index - 1];

        if (!point) return;

        const value = point.ExecutionMetrics[metricKey];

        // Skip if value is undefined or NaN
        if (value === undefined || isNaN(value)) return;

        const color = s.color || d3.schemeCategory10[i % 10];
        const xPos = x(point.BlockNumber);
        const yPos = y(value);

        // Skip if position is invalid
        if (isNaN(xPos) || isNaN(yPos)) return;

        if (svgRef.current) {
          // Create a temporary text element to measure size
          const tempText = d3
            .select(svgRef.current)
            .append("text")
            .attr("font-size", "10px")
            .attr("font-family", "sans-serif")
            .text(`${s.name}: ${formatValue(value, unit)}`)
            .attr("visibility", "hidden");

          const textBox = (tempText.node() as SVGTextElement).getBBox();
          tempText.remove();

          // For safety, check that dimensions are valid
          if (textBox.width > 0 && textBox.height > 0) {
            newTooltips.push({
              id: i,
              x: xPos, // X position of the data point
              y: yPos - textBox.height - 10, // Initial position above the point
              width: textBox.width + 10,
              height: textBox.height + 6,
              value: value,
              originalY: yPos - textBox.height - 10,
              seriesName: s.name,
              formattedValue: formatValue(value, unit),
              color: color,
            });
          }
        }
      });

      return newTooltips;
    },
    [series, metricKey, unit],
  );

  // Function to update hover elements
  const updateHoverDisplay = useCallback(
    (tooltips: TooltipData[], mouseX: number) => {
      if (
        !hoverGroupRef.current ||
        !tooltipContainerRef.current ||
        !verticalLineRef.current
      )
        return;

      // Show the hover group
      hoverGroupRef.current.style("display", null);

      // Update vertical line using exact pixel position
      verticalLineRef.current.attr("x1", mouseX).attr("x2", mouseX);

      // Clear existing tooltips
      const container = tooltipContainerRef.current;
      container.selectAll("*").remove();

      // Skip if no tooltips to show
      if (!tooltips || tooltips.length === 0) return;

      // Add tooltips with adjusted positions
      tooltips.forEach((tooltip) => {
        // Skip invalid positions
        if (isNaN(tooltip.x) || isNaN(tooltip.y)) return;

        // Add tooltip group
        const group = container.append("g");

        // Add dot at data point
        group
          .append("circle")
          .attr("r", 5)
          .attr("cx", tooltip.x)
          .attr("cy", tooltip.originalY + tooltip.height + 5)
          .attr("fill", tooltip.color);

        // Add tooltip background
        group
          .append("rect")
          .attr("rx", 3)
          .attr("ry", 3)
          .attr("x", tooltip.x - tooltip.width / 2)
          .attr("y", tooltip.y)
          .attr("width", tooltip.width)
          .attr("height", tooltip.height)
          .attr("fill", "rgba(255, 255, 255, 0.9)")
          .attr("stroke", tooltip.color)
          .attr("stroke-width", 1);

        // Add tooltip text
        group
          .append("text")
          .attr("x", tooltip.x)
          .attr("y", tooltip.y + tooltip.height / 2 + 3)
          .attr("font-size", "10px")
          .attr("font-family", "sans-serif")
          .attr("fill", "#333")
          .attr("text-anchor", "middle")
          .text(`${tooltip.seriesName}: ${tooltip.formattedValue}`);
      });
    },
    [],
  );

  // Handle hover events from any chart
  const handleHover = useCallback(
    (event: ChartHoverEvent) => {
      if (!chartRef.current) return;

      // Store mouseX in ref instead of state
      mouseXRef.current = event.mouseX;

      // Calculate tooltips based on pixel position
      const newTooltips = calculateTooltips(event.mouseX);

      // Only update state if tooltips have changed, using lodash's isEqual for deep comparison
      if (!isEqual(newTooltips, tooltips)) {
        setTooltips(newTooltips);
      }

      // Show immediate feedback with just the vertical line
      if (hoverGroupRef.current && verticalLineRef.current) {
        // Make hover group visible
        hoverGroupRef.current.style("display", null);

        // Update vertical line position
        verticalLineRef.current
          .attr("x1", event.mouseX)
          .attr("x2", event.mouseX);
      }
    },
    [calculateTooltips, tooltips],
  );

  // Handle hover end
  const handleHoverEnd = useCallback(() => {
    if (hoverGroupRef.current) {
      hoverGroupRef.current.style("display", "none");
    }
    mouseXRef.current = null;
    setTooltips([]);
  }, []);

  // Sync tooltip display when adjusted tooltips change
  useEffect(() => {
    if (mouseXRef.current !== null && adjustedTooltips.length > 0) {
      // Update with properly positioned tooltips
      updateHoverDisplay(adjustedTooltips, mouseXRef.current);
    }
  }, [adjustedTooltips, updateHoverDisplay]);

  // Use our shared hover hook
  const { triggerHover, triggerHoverEnd } = useChartHoverSync(
    chartId,
    handleHover,
    handleHoverEnd,
  );

  return (
    <BaseChart
      data={validData}
      metricKey={metricKey}
      title={title}
      description={description}
    >
      {(svg, dimensions) => {
        // Store refs for use in effects and callbacks
        svgRef.current = svg.node();
        chartRef.current = dimensions;

        // Create scales based on filtered valid data
        const x = d3
          .scaleLinear()
          .domain([minBlock, maxBlock])
          .range([0, dimensions.width]);

        const y = d3
          .scaleLinear()
          .domain([0, maxValue > 0 ? maxValue : 1]) // Ensure non-zero domain to avoid NaN
          .range([dimensions.height, 0]);

        // Store scales in refs for hover calculations
        xScaleRef.current = x;
        yScaleRef.current = y;

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
          .call(
            d3
              .axisBottom(x)
              .ticks(maxBlock - minBlock)
              .tickFormat((d) => (Number.isInteger(d) ? d.toString() : "")),
          )
          .selectAll("text")
          .style("text-anchor", "end")
          .attr("dx", "-.8em")
          .attr("dy", ".15em")
          .attr("transform", "rotate(-45)");

        // Add x-axis label if provided
        if (xAxisLabel) {
          svg
            .append("text")
            .attr("class", "x-axis-label")
            .attr("text-anchor", "middle")
            .attr("x", dimensions.width / 2)
            .attr("y", dimensions.height + 27)
            .attr("font-size", 12)
            .attr("fill", "#333")
            .text(xAxisLabel);
        }

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
          .text((d) => {
            return formatLabel(d.name);
          });

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
            `translate(${currentX}, ${dimensions.height + 36})`,
          ); // Move legend closer
          currentX += legendItemWidths[i] + spacing; // Add spacing between items
        });

        series.forEach((s, i) => {
          const color = s.color || d3.schemeCategory10[i % 10];

          // Filter out data points with undefined or NaN values for this series
          const seriesValidData = s.data.filter((d) => {
            const val = d.ExecutionMetrics[metricKey];
            return val !== undefined && !isNaN(val);
          });

          // Only draw line if we have valid data
          if (seriesValidData.length > 0) {
            // Add the line
            const line = d3
              .line<MetricData>()
              .defined((d) => {
                const val = d.ExecutionMetrics[metricKey];
                return val !== undefined && !isNaN(val);
              })
              .x((d) => x(d.BlockNumber))
              .y((d) => {
                const val = d.ExecutionMetrics[metricKey];
                return y(val);
              });

            svg
              .append("path")
              .datum(seriesValidData)
              .attr("fill", "none")
              .attr("stroke", color)
              .attr("stroke-width", 2)
              .attr("d", line);

            // Add dots
            svg
              .selectAll(`.dot-${i}`)
              .data(seriesValidData)
              .enter()
              .append("circle")
              .attr("class", `dot-${i}`)
              .attr("cx", (d) => x(d.BlockNumber))
              .attr("cy", (d) => y(d.ExecutionMetrics[metricKey]))
              .attr("r", 4)
              .style("fill", color)
              .style("opacity", 0);
          }
        });

        // Add hover elements
        const hoverGroup = svg
          .append("g")
          .attr("class", "hover-elements")
          .style("display", "none");
        hoverGroupRef.current = hoverGroup;

        // Add vertical line
        const verticalLine = hoverGroup
          .append("line")
          .attr("class", "hover-line")
          .attr("y1", 0)
          .attr("y2", dimensions.height)
          .attr("stroke", "#666")
          .attr("stroke-width", 1)
          .attr("stroke-dasharray", "3,3");
        verticalLineRef.current = verticalLine;

        // Create tooltip container and store ref
        const tooltipContainer = hoverGroup
          .append("g")
          .attr("class", "tooltips-container");
        tooltipContainerRef.current = tooltipContainer;

        // Create mouse tracking overlay using DOM events
        svg
          .append("rect")
          .attr("class", "overlay")
          .attr("width", dimensions.width)
          .attr("height", dimensions.height)
          .attr("fill", "none")
          .attr("pointer-events", "all")
          .on("mouseover", () => {
            hoverGroup.style("display", null);
          })
          .on("mouseout", () => {
            hoverGroup.style("display", "none");
            triggerHoverEnd();
          })
          .on("mousemove", function (event) {
            const [mouseX] = d3.pointer(event, this);
            triggerHover(mouseX);
          });
      }}
    </BaseChart>
  );
};

export default LineChart;
