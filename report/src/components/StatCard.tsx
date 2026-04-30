import { ReactNode } from "react";
import clsx from "clsx";

interface StatCardProps {
  title: string;
  children: ReactNode;
  className?: string;
}

const StatCard = ({ title, children, className }: StatCardProps) => {
  return (
    <section
      className={clsx(
        "border border-slate-200 bg-white rounded-lg p-6",
        className,
      )}
    >
      <h2 className="text-sm font-semibold text-slate-500 uppercase tracking-wide mb-4">
        {title}
      </h2>
      {children}
    </section>
  );
};

export default StatCard;

interface StatProps {
  label: string;
  value: ReactNode;
  hint?: string;
}

export const Stat = ({ label, value, hint }: StatProps) => (
  <div className="flex flex-col">
    <div className="text-xs text-slate-500">{label}</div>
    <div className="text-xl font-medium text-slate-900 tabular-nums">
      {value}
    </div>
    {hint && <div className="text-xs text-slate-400 mt-1">{hint}</div>}
  </div>
);

export const StatGrid = ({ children }: { children: ReactNode }) => (
  <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-6">
    {children}
  </div>
);
