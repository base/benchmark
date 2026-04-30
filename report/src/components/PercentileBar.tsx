import { ReactNode } from "react";
import clsx from "clsx";

export interface PercentileBarRow {
  label: string;
  numericValue: number;
  display: ReactNode;
  emphasized?: boolean;
}

interface PercentileBarChartProps {
  rows: PercentileBarRow[];
  barColorClass?: string;
}

const PercentileBarChart = ({
  rows,
  barColorClass = "bg-blue-500",
}: PercentileBarChartProps) => {
  const max = rows.reduce((m, r) => Math.max(m, r.numericValue), 0);

  return (
    <div className="flex flex-col gap-y-2">
      {rows.map((row) => {
        const pct = max === 0 ? 0 : (row.numericValue / max) * 100;
        return (
          <div
            key={row.label}
            className="grid grid-cols-[3rem_1fr_auto] items-center gap-x-3"
          >
            <div
              className={clsx(
                "text-xs text-slate-500 font-mono",
                row.emphasized && "font-semibold text-slate-900",
              )}
            >
              {row.label}
            </div>
            <div className="bg-slate-100 rounded h-2 relative overflow-hidden">
              <div
                className={clsx("h-2 rounded transition-all", barColorClass)}
                style={{ width: `${pct}%` }}
              />
            </div>
            <div
              className={clsx(
                "text-sm text-slate-700 tabular-nums min-w-[6rem] text-right",
                row.emphasized && "font-semibold text-slate-900",
              )}
            >
              {row.display}
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default PercentileBarChart;
