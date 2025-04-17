import React, { useRef, useEffect, useState } from "react";
import * as d3 from "d3";
import { MetricData } from "../types";

interface BaseChartProps {
  data: MetricData[];
  metricKey: string;
  title?: string;
  description?: string;
  children: (
    svg: d3.Selection<SVGGElement, unknown, null, undefined>,
    dimensions: {
      width: number;
      height: number;
      margin: { top: number; right: number; bottom: number; left: number };
    },
  ) => void;
}

const TOP_MARGIN = 20;
const ASPECT_RATIO = 0.5;
const LEGEND_SPACE = 20;
const X_AXIS_SPACE = 60;
const Y_AXIS_SPACE = 40;
const TITLE_SPACE = 50;

const DEFAULT_MARGIN = {
  top: TOP_MARGIN + TITLE_SPACE,
  right: 40,
  bottom: Y_AXIS_SPACE + LEGEND_SPACE,
  left: X_AXIS_SPACE,
};

const BaseChart: React.FC<BaseChartProps> = ({
  data,
  metricKey,
  title,
  description,
  children,
}: BaseChartProps) => {
  const svgRef = useRef<SVGSVGElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [dimensions, setDimensions] = useState<{
    width: number;
    height: number;
    margin: typeof DEFAULT_MARGIN;
  } | null>(null);

  const initialDimensions = useRef(dimensions);
  initialDimensions.current = dimensions;

  useEffect(() => {
    const updateDimensions = () => {
      if (wrapperRef.current) {
        const width = wrapperRef.current.offsetWidth;
        const height = width * ASPECT_RATIO;
        // Ensure minimum dimensions after margin calculation
        const calculatedWidth = Math.max(
          100,
          width - DEFAULT_MARGIN.left - DEFAULT_MARGIN.right,
        );
        const calculatedHeight = Math.max(
          50,
          height - DEFAULT_MARGIN.top - DEFAULT_MARGIN.bottom,
        );
        setDimensions({
          width: calculatedWidth,
          height: calculatedHeight,
          margin: DEFAULT_MARGIN,
        });
      }
    };

    // Ensure initial dimensions are set even if ResizeObserver fires quickly
    if (!initialDimensions.current && wrapperRef.current?.offsetWidth) {
      updateDimensions();
    }

    const resizeObserver = new ResizeObserver(updateDimensions);
    if (wrapperRef.current) {
      resizeObserver.observe(wrapperRef.current);
    }

    return () => {
      resizeObserver.disconnect();
    };
    // Only depends on the existence of the wrapper ref
  }, []);

  useEffect(() => {
    if (!svgRef.current || !dimensions) return;

    const svgRoot = d3.select(svgRef.current);
    svgRoot.selectAll("*").remove(); // Clear previous render

    const svg = svgRoot
      .attr(
        "width",
        dimensions.width + dimensions.margin.left + dimensions.margin.right,
      )
      .attr(
        "height",
        dimensions.height + dimensions.margin.top + dimensions.margin.bottom,
      )
      .append("g")
      .attr(
        "transform",
        `translate(${dimensions.margin.left},${dimensions.margin.top})`,
      );

    if (title) {
      svg
        .append("text")
        .attr("x", dimensions.width / 2)
        .attr("y", -TOP_MARGIN - 20)
        .attr("text-anchor", "middle")
        .style("font-size", "16px")
        .style("font-weight", "bold")
        .text(title);
    }

    if (description) {
      svg
        .append("text")
        .attr("x", dimensions.width / 2)
        .attr("y", -TOP_MARGIN)
        .attr("text-anchor", "middle")
        .style("font-size", "12px")
        .style("fill", "#666")
        .text(description);
    }

    children(svg, dimensions);

    // Re-render when data or dimensions change
  }, [data, metricKey, title, description, children, dimensions]);

  return (
    <div className="chart-wrapper" ref={wrapperRef}>
      {/* Render SVG only when dimensions are known */}
      {dimensions && <svg ref={svgRef}></svg>}
    </div>
  );
};

export default BaseChart;
