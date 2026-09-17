interface TooltipPosition {
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
 * Function to calculate non-overlapping positions for tooltips
 * @param positions Array of tooltip positions with dimensions
 * @param chartHeight Total height of the chart area
 * @returns Array of adjusted positions
 */
export const calculateTooltipLayout = (
  positions: TooltipPosition[],
  chartHeight: number,
): TooltipPosition[] => {
  if (!positions.length) return [];

  // Make a copy to avoid mutating the original
  const result = positions.map((pos) => ({ ...pos }));

  // Sort by y position (top to bottom)
  result.sort((a, b) => a.originalY - b.originalY);

  // Simple algorithm to prevent overlaps
  // We'll do a single pass over the sorted tooltips
  for (let i = 1; i < result.length; i++) {
    const current = result[i];
    const prev = result[i - 1];

    // Check if tooltips overlap
    const minSpace = 5; // Minimum pixels between tooltips
    const overlap = prev.y + prev.height + minSpace > current.y;

    if (overlap) {
      // Position current tooltip below previous one with spacing
      current.y = prev.y + prev.height + minSpace;
    }
  }

  // Check if any tooltips go off the bottom of the chart
  // and reposition them above their data point if needed
  for (let i = 0; i < result.length; i++) {
    const tooltip = result[i];

    if (tooltip.y + tooltip.height > chartHeight) {
      // Calculate position above the data point
      const posAbove = tooltip.originalY - tooltip.height - 10;

      // Only move it up if there's room and it doesn't overlap with previous tooltips
      if (posAbove > 0) {
        let canMoveUp = true;

        // Check for overlaps with tooltips already positioned above
        for (let j = 0; j < i; j++) {
          const above = result[j];
          if (
            above.y < tooltip.originalY && // Only check tooltips above the data point
            above.y + above.height + 5 > posAbove
          ) {
            canMoveUp = false;
            break;
          }
        }

        if (canMoveUp) {
          tooltip.y = posAbove;
        }
      }
    }
  }

  return result;
};

export default calculateTooltipLayout;
