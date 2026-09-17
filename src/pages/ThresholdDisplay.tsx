import React from "react";

interface ThresholdDisplayProps {
  children?: React.ReactNode;
  value: number;
  warningThreshold?: number;
  errorThreshold?: number;
}

const ThresholdDisplay = ({
  value,
  warningThreshold,
  errorThreshold,
  children,
}: ThresholdDisplayProps) => {
  const isWarning = warningThreshold && value > warningThreshold;
  const isError = errorThreshold && value > errorThreshold;

  return (
    <div
      className={`${isError ? "text-red-600 font-bold" : isWarning ? "text-yellow-600 font-semibold" : ""}`}
    >
      {children}
    </div>
  );
};

export default ThresholdDisplay;
