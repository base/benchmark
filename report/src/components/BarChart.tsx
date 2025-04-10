import React from 'react'
import * as d3 from 'd3'
import { DataSeries, MetricData, ChartConfig } from '../types'
import BaseChart from './BaseChart'
import { formatValue } from '../utils/formatters'; // Import shared formatter

interface BarChartProps {
  series: DataSeries[]
  metricKey: string
  title?: string
  description?: string
  unit?: ChartConfig['unit']
}

const BarChart: React.FC<BarChartProps> = ({ series, metricKey, title, description, unit }) => {
  // Get all data points for domain calculation
  const allData = series.flatMap(s => s.data)
  
  return (
    <BaseChart data={allData} metricKey={metricKey} title={title} description={description}>
      {(svg, dimensions) => {
        // Create an array of all block numbers
        const blockNumbers = allData.map(d => d.BlockNumber)
        const minBlock = Math.min(...blockNumbers)
        const maxBlock = Math.max(...blockNumbers)
        
        const x = d3.scaleLinear()
          .domain([minBlock, maxBlock])
          .range([0, dimensions.width])

        const y = d3.scaleLinear()
          .domain([0, d3.max(allData, d => d.ExecutionMetrics[metricKey]) as number])
          .range([dimensions.height, 0])

        // Add grid lines
        svg.append('g')
          .attr('class', 'grid')
          .attr('transform', `translate(0,${dimensions.height})`)
          .call(d3.axisBottom(x)
            .tickSize(-dimensions.height)
            .tickFormat(() => '')
          )
          .style('stroke-dasharray', '3,3')
          .style('stroke-opacity', 0.2)

        svg.append('g')
          .attr('class', 'grid')
          .call(d3.axisLeft(y)
            .tickSize(-dimensions.width)
            .tickFormat(() => '')
          )
          .style('stroke-dasharray', '3,3')
          .style('stroke-opacity', 0.2)

        // Add axes
        svg.append('g')
          .attr('transform', `translate(0,${dimensions.height})`)
          .call(d3.axisBottom(x))
          .selectAll('text')
          .style('text-anchor', 'end')
          .attr('dx', '-.8em')
          .attr('dy', '.15em')
          .attr('transform', 'rotate(-45)')

        svg.append('g')
          .call(d3.axisLeft(y).tickFormat(d => formatValue(d as number, unit)))
          .append('text')
          .attr('fill', '#000')
          .attr('transform', 'rotate(-90)')
          .attr('y', 6)
          .attr('dy', '.71em')
          .style('text-anchor', 'end');

        // Calculate bar width based on number of blocks and series
        const barWidth = Math.max(1, Math.min(20, dimensions.width / (maxBlock - minBlock + 1) / series.length))

        // Add legend
        const legend = svg.append('g')
          .attr('class', 'legend')
          .attr('font-family', 'sans-serif')
          .attr('font-size', 10)
          .attr('text-anchor', 'start')
          .selectAll('g')
          .data(series)
          .join('g');
          // Position will be set after calculating width

        legend.append('rect')
          .attr('x', 0)
          .attr('width', 10) // Smaller square
          .attr('height', 10) // Smaller square
          .attr('fill', (d, i) => d.color || d3.schemeCategory10[i % 10]);

        legend.append('text')
          .attr('x', 15) // Adjust text position
          .attr('y', 5)  // Adjust text position (center vertically)
          .attr('dy', '0.35em')
          .text(d => d.name);

        // Center the legend group horizontally
        const legendGroupSelection = svg.selectAll('.legend > g') as d3.Selection<SVGGElement, DataSeries, SVGGElement, unknown>;
        let totalLegendWidth = 0;
        const legendItemWidths: number[] = [];
        
        legendGroupSelection.each(function() {
          const bbox = (this as SVGGraphicsElement).getBBox();
          legendItemWidths.push(bbox.width);
          totalLegendWidth += bbox.width;
        });

        const spacing = 10;
        totalLegendWidth += Math.max(0, series.length - 1) * spacing; // Add spacing between items
        
        const startX = Math.max(0, (dimensions.width - totalLegendWidth) / 2);
        let currentX = startX;

        legendGroupSelection.each(function(d, i) {
            d3.select(this).attr('transform', `translate(${currentX}, ${dimensions.height + 25})`); // Move legend closer
            currentX += legendItemWidths[i] + spacing; // Add spacing between items
        });

        series.forEach((s, i) => {
          const color = s.color || d3.schemeCategory10[i % 10]

          // Add bars
          const bars = svg.selectAll(`.bar-${i}`)
            .data(s.data)
            .enter()
            .append('rect')
            .attr('class', `bar-${i}`)
            .attr('x', d => x(d.BlockNumber) - (barWidth * series.length / 2) + (i * barWidth))
            .attr('y', d => y(d.ExecutionMetrics[metricKey]))
            .attr('width', barWidth)
            .attr('height', d => dimensions.height - y(d.ExecutionMetrics[metricKey]))
            .style('fill', color)
            .style('opacity', 0.8)

          // Add hover effects (Need to add back if you want tooltips)
          bars
            .on('mouseover', (event, d) => {
              d3.select(event.currentTarget)
                .transition()
                .duration(200)
                .style('opacity', 1)
    
              const tooltip = d3.select('.tooltip')
              tooltip
                .style('opacity', 1)
                // Use formatValue for the tooltip
                .html(`${s.name}<br>Block: ${d.BlockNumber}<br>Value: ${formatValue(d.ExecutionMetrics[metricKey], unit)}`)
                .style('left', (event.pageX + 10) + 'px')
                .style('top', (event.pageY - 28) + 'px')
            })
            .on('mouseout', (event) => {
              d3.select(event.currentTarget)
                .transition()
                .duration(200)
                .style('opacity', 0.8)
    
              d3.select('.tooltip').style('opacity', 0)
            })
        })
      }}
    </BaseChart>
  )
}

export default BarChart 