import { useState } from "react";
import { ChevronDownIcon } from "@heroicons/react/24/outline";

interface CollapsibleProps {
  title: React.ReactNode;
  defaultCollapsed?: boolean;
  children: React.ReactNode;
}

const Collapsible = ({
  title,
  defaultCollapsed = false,
  children,
}: CollapsibleProps) => {
  const [collapsed, setCollapsed] = useState(defaultCollapsed);

  return (
    <div>
      <div className="border border-slate-200 rounded-lg bg-white cursor-pointer">
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="flex items-center justify-between text-left p-4 w-full"
        >
          <div className="font-medium flex-grow">{title}</div>
          <div className="text-gray-500 flex-shrink-0">
            <ChevronDownIcon
              className="w-4 h-4"
              style={{
                transform: collapsed ? "rotate(180deg)" : "rotate(0deg)",
              }}
            />
          </div>
        </button>

        {!collapsed && (
          <div className="p-4 text-gray-800 prose-sm prose border-t border-t-slate-200">
            {children}
          </div>
        )}
      </div>
    </div>
  );
};

export default Collapsible;
