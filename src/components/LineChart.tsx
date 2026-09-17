import React, {
  useRef,
  useEffect,
  useCallback,
  useId,
  useState,
  useMemo,
} from "react";
import * as d3 from "d3";
import { DataSeries, MetricData, ChartConfig } from "../types";
import BaseChart from "./BaseChart";
import { formatValue } from "../utils/formatters";
import calculateTooltipLayout from "../hooks/useTooltipLayout";
import useChartHoverSync, { ChartHoverEvent } from "../hooks/useChartHoverSync";
import { isEqual } from "lodash";

interface LineChartProps {
  series: DataSeries[];
  thresholdKey: string | null;
  metricKey: string;
  title?: string;
  description?: string;
  unit?: ChartConfig["unit"];
  xAxisDomain?: [number, number];
  xAxisLabel?: string;
  thresholds?: {
    warning?: Record<string, number>;
    error?: Record<string, number>;
  };
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

/**
 * Max ticks on the x axis.
 */
const MAX_TICKS = 10;

/** Never render more than this many averaged samples for one series. */
const MAX_RENDERED_POINTS = 640;

const CHART_RATE_PREFIXES = [
  ["E", 1e18],
  ["P", 1e15],
  ["T", 1e12],
  ["G", 1e9],
  ["M", 1e6],
  ["k", 1e3],
  ["", 1],
] as const;

const sampleX = (data: MetricData): number =>
  data.PercentComplete ?? data.ElapsedMilliseconds ?? data.BlockNumber;

const formatElapsedTime = (milliseconds: number, unit: "ms" | "s"): string => {
  if (unit === "s") {
    const seconds = milliseconds / 1_000;
    return `${Number.isInteger(seconds) ? seconds : seconds.toFixed(1)}s`;
  }
  return `${Math.round(milliseconds)}ms`;
};

// Axis and hover labels need to remain compact even when a chart's scale is
// very large. This intentionally does not affect table cells or downloads.
const formatChartValue = (value: number, unit: ChartConfig["unit"]): string => {
  if (unit !== "gas/s" || !Number.isFinite(value)) {
    return formatValue(value, unit);
  }

  if (value === 0) return "0 gas/s";
  const prefixIndex = CHART_RATE_PREFIXES.findIndex(
    ([, multiplier]) => Math.abs(value) >= multiplier,
  );
  const selectedIndex =
    prefixIndex === -1 ? CHART_RATE_PREFIXES.length - 1 : prefixIndex;
  const [prefix, multiplier] = CHART_RATE_PREFIXES[selectedIndex];
  const scaledValue = value / multiplier;

  if (Math.abs(scaledValue) >= 100 && selectedIndex > 0) {
    const [largerPrefix, largerMultiplier] =
      CHART_RATE_PREFIXES[selectedIndex - 1];
    return `${(value / largerMultiplier).toFixed(1)} ${largerPrefix}gas/s`;
  }

  return `${scaledValue.toFixed(1)} ${prefix}gas/s`;
};

/**
 * Reduces a time series to contiguous buckets, preserving the average x and y
 * value of each bucket. This makes dense, spiky benchmark metrics legible
 * without drawing more than one point per two horizontal pixels.
 */
const averageSamples = (
  data: MetricData[],
  metricKey: string,
  maxPoints: number,
): MetricData[] => {
  const validSamples = data.filter((sample) => {
    const value = sample.ExecutionMetrics[metricKey];
    return typeof value === "number" && Number.isFinite(value);
  });

  if (validSamples.length <= maxPoints) return validSamples;

  const buckets = Math.min(maxPoints, validSamples.length);
  return Array.from({ length: buckets }, (_, bucketIndex) => {
    const start = Math.floor((bucketIndex * validSamples.length) / buckets);
    const end = Math.floor(((bucketIndex + 1) * validSamples.length) / buckets);
    const samples = validSamples.slice(start, Math.max(start + 1, end));
    const averageX = d3.mean(samples, sampleX) ?? 0;
    const averageElapsedMilliseconds =
      d3.mean(
        samples,
        (sample) => sample.ElapsedMilliseconds ?? sample.BlockNumber,
      ) ?? 0;
    const hasPercentComplete = samples.some(
      (sample) =>
        typeof sample.PercentComplete === "number" &&
        Number.isFinite(sample.PercentComplete),
    );
    const averageValue =
      d3.mean(samples, (sample) => sample.ExecutionMetrics[metricKey]) ?? 0;

    return {
      ...samples[0],
      BlockNumber: averageElapsedMilliseconds,
      ElapsedMilliseconds: averageElapsedMilliseconds,
      PercentComplete: hasPercentComplete ? averageX : undefined,
      ExecutionMetrics: {
        [metricKey]: averageValue,
      },
    };
  });
};

const LineChart: React.FC<LineChartProps> = ({
  series,
  thresholdKey,
  metricKey,
  title,
  description,
  unit,
  xAxisDomain,
  xAxisLabel,
  thresholds,
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

  // Calculate adjusted tooltip positions using the pure function
  const adjustedTooltips = calculateTooltipLayout(
    tooltips,
    chartRef.current?.height || 0,
  );

  const renderedSeriesRef = useRef<DataSeries[]>(series);

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

      renderedSeriesRef.current.forEach((s, i) => {
        // Skip if series has no data
        if (!s.data.length) return;

        // Find the closest point in the series data
        const bisect = d3.bisector(sampleX).left;
        const index = bisect(s.data, xValue);

        // Handle edge cases
        const point =
          index >= s.data.length
            ? s.data[s.data.length - 1]
            : index <= 0
              ? s.data[0]
              : Math.abs(sampleX(s.data[index]) - xValue) <
                  Math.abs(sampleX(s.data[index - 1]) - xValue)
                ? s.data[index]
                : s.data[index - 1];

        if (!point) return;

        const value = point.ExecutionMetrics[metricKey];

        // Skip if value is undefined or NaN
        if (value === undefined || isNaN(value)) return;

        const color = s.color || d3.schemeCategory10[i % 10];
        const xPos = x(sampleX(point));
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
            .text(`${s.name}: ${formatChartValue(value, unit)}`)
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
              formattedValue: formatChartValue(value, unit),
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

  const maxThreshold = useMemo(() => {
    let maxThreshold = 0;
    for (const thresholdMap of [thresholds?.warning, thresholds?.error]) {
      if (thresholdMap && thresholdKey) {
        if (thresholdMap[thresholdKey] > maxThreshold) {
          maxThreshold = thresholdMap[thresholdKey];
        }
      }
    }
    return maxThreshold;
  }, [thresholds, thresholdKey]);

  return (
    <BaseChart
      data={allData}
      metricKey={metricKey}
      title={title}
      description={description}
    >
      {(svg, dimensions) => {
        const maxPoints = Math.max(
          1,
          Math.min(MAX_RENDERED_POINTS, Math.floor(dimensions.width / 2)),
        );
        const renderedSeries = series.map((item) => ({
          ...item,
          data: averageSamples(item.data, metricKey, maxPoints),
        }));
        renderedSeriesRef.current = renderedSeries;
        const renderedData = renderedSeries.flatMap((item) => item.data);
        const usePercentComplete = renderedData.some(
          (sample) =>
            typeof sample.PercentComplete === "number" &&
            Number.isFinite(sample.PercentComplete),
        );
        const blockNumbers = renderedData.map(sampleX);
        const minBlock = xAxisDomain
          ? xAxisDomain[0]
          : usePercentComplete
            ? 0
            : blockNumbers.length
              ? Math.min(...blockNumbers)
              : 0;
        const maxBlock = xAxisDomain
          ? xAxisDomain[1]
          : usePercentComplete
            ? 100
            : blockNumbers.length
              ? Math.max(...blockNumbers)
              : 100;
        const maxValue =
          (d3.max(
            renderedData,
            (d) => d.ExecutionMetrics[metricKey],
          ) as number) || 0;
        const elapsedTimeUnit =
          !usePercentComplete && maxBlock >= 1_000 ? "s" : "ms";
        const resolvedXAxisLabel =
          xAxisLabel ??
          (usePercentComplete
            ? "Benchmark completion"
            : `Elapsed Time (${elapsedTimeUnit})`);

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
          .domain([0, maxValue > 0 ? Math.max(maxValue, maxThreshold) : 1]) // Ensure non-zero domain to avoid NaN
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

        // Add threshold lines if thresholds are provided
        if (thresholds) {
          // Add warning threshold line (yellow)
          if (
            thresholds.warning &&
            thresholdKey &&
            thresholds.warning[thresholdKey] !== undefined
          ) {
            const warningY = y(thresholds.warning[thresholdKey]);
            svg
              .append("line")
              .attr("class", "threshold-line warning")
              .attr("x1", 0)
              .attr("x2", dimensions.width)
              .attr("y1", warningY)
              .attr("y2", warningY)
              .attr("stroke", "#ffc107") // Yellow color
              .attr("stroke-dasharray", "3,4")
              .style("opacity", 0.8);
          }

          // Add error threshold line (red)
          if (
            thresholds.error &&
            thresholdKey &&
            thresholds.error[thresholdKey] !== undefined
          ) {
            const errorY = y(thresholds.error[thresholdKey]);
            svg
              .append("line")
              .attr("class", "threshold-line error")
              .attr("x1", 0)
              .attr("x2", dimensions.width)
              .attr("y1", errorY)
              .attr("y2", errorY)
              .attr("stroke", "#dc3545") // Red color
              .attr("stroke-width", 1)
              .attr("stroke-dasharray", "6,4")
              .style("opacity", 0.8);
          }
        }

        // Add axes
        svg
          .append("g")
          .attr("transform", `translate(0,${dimensions.height})`)
          .call(
            d3
              .axisBottom(x)
              .ticks(Math.min(maxBlock - minBlock, MAX_TICKS))
              .tickFormat((d) =>
                usePercentComplete
                  ? `${Math.round(d as number)}%`
                  : formatElapsedTime(d as number, elapsedTimeUnit),
              ),
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
            .attr("y", dimensions.height + 40)
            .attr("font-size", 12)
            .attr("fill", "#333")
            .text(resolvedXAxisLabel);
        }

        svg
          .append("g")
          .call(
            d3
              .axisLeft(y)
              .tickFormat((d) => formatChartValue(d as number, unit))
              .ticks(8),
          )
          .append("text")
          .attr("fill", "#000")
          .attr("transform", "rotate(-90)")
          .attr("y", 6)
          .attr("dy", ".71em")
          .style("text-anchor", "end");

        renderedSeries.forEach((s, i) => {
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
              .x((d) => x(sampleX(d)))
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
              .attr("cx", (d) => x(sampleX(d)))
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
