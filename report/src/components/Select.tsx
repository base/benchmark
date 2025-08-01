import React from "react";
import clsx from "clsx";

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  children: React.ReactNode;
  className?: string;
}

const Select = ({ children, className, ...props }: SelectProps) => {
  return (
    <select
      className={clsx(
        "bg-white border border-slate-300 rounded-md px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500",
        className,
      )}
      {...props}
    >
      {children}
    </select>
  );
};

export default Select;
