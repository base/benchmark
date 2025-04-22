/**
 * A custom hook to handle synchronized hover effects across multiple charts.
 * Uses a pub/sub pattern with DOM custom events to prevent unnecessary re-renders.
 */

import { useEffect, useRef } from "react";

// Define type for the hover event's detail
export interface ChartHoverEvent {
  mouseX: number; // X position in pixels
  chartId: string; // Unique ID for the chart that triggered the event
}

// Event name for chart hover events
const CHART_HOVER_EVENT = "chart-hover";
const CHART_HOVER_END_EVENT = "chart-hover-end";

/**
 * Hook to create and manage chart hover synchronization
 * @param chartId Unique identifier for this chart instance
 * @param onHover Callback when hover happens on any chart
 * @param onHoverEnd Callback when hover ends on any chart
 */
export const useChartHoverSync = (
  chartId: string,
  onHover: (event: ChartHoverEvent) => void,
  onHoverEnd: () => void,
) => {
  // Keep refs to callbacks to avoid creating new event listeners on re-renders
  const onHoverRef = useRef(onHover);
  const onHoverEndRef = useRef(onHoverEnd);

  // Update refs when callbacks change
  useEffect(() => {
    onHoverRef.current = onHover;
    onHoverEndRef.current = onHoverEnd;
  }, [onHover, onHoverEnd]);

  // Set up event listeners for global hover events
  useEffect(() => {
    // Handler for hover events from any chart
    const handleHover = (e: Event) => {
      const customEvent = e as CustomEvent<ChartHoverEvent>;
      onHoverRef.current(customEvent.detail);
    };

    // Handler for hover end events
    const handleHoverEnd = () => {
      onHoverEndRef.current();
    };

    // Add global event listeners
    document.addEventListener(CHART_HOVER_EVENT, handleHover);
    document.addEventListener(CHART_HOVER_END_EVENT, handleHoverEnd);

    // Clean up listeners on unmount
    return () => {
      document.removeEventListener(CHART_HOVER_EVENT, handleHover);
      document.removeEventListener(CHART_HOVER_END_EVENT, handleHoverEnd);
    };
  }, []);

  // Return functions to trigger hover events
  return {
    // Call this when the mouse moves over this chart
    triggerHover: (mouseX: number) => {
      const event = new CustomEvent<ChartHoverEvent>(CHART_HOVER_EVENT, {
        detail: {
          mouseX,
          chartId,
        },
        bubbles: true,
      });
      document.dispatchEvent(event);
    },

    // Call this when the mouse leaves this chart
    triggerHoverEnd: () => {
      const event = new CustomEvent(CHART_HOVER_END_EVENT, {
        bubbles: true,
      });
      document.dispatchEvent(event);
    },
  };
};

export default useChartHoverSync;
