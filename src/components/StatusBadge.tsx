interface StatusBadgeProps {
  status: string;
  showCount?: boolean;
  count?: number;
  className?: string;
}

// Helper function to get status badge styling
const getStatusBadgeStyle = (status: string) => {
  switch (status) {
    case "success":
      return "bg-emerald-50 text-emerald-700 ring-emerald-600/20";
    case "error":
      return "bg-red-50 text-red-700 ring-red-600/20";
    case "warning":
      return "bg-amber-50 text-amber-700 ring-amber-600/20";
    case "incomplete":
      return "bg-amber-50 text-amber-700 ring-amber-600/20";
    default:
      return "bg-slate-50 text-slate-700 ring-slate-600/20";
  }
};

// Helper function to get status display text
const getStatusDisplayText = (status: string) => {
  switch (status) {
    case "incomplete":
      return "In Progress";
    case "success":
      return "Success";
    case "warning":
      return "Warning";
    case "error":
      return "Error";
    default:
      return status;
  }
};

// Helper function to get status indicator color
const getStatusIndicatorColor = (status: string) => {
  switch (status) {
    case "success":
      return "bg-emerald-500";
    case "error":
      return "bg-red-500";
    case "warning":
    case "incomplete":
      return "bg-amber-500";
    default:
      return "bg-slate-500";
  }
};

const StatusBadge = ({
  status,
  showCount = false,
  count,
  className = "",
}: StatusBadgeProps) => {
  const badgeStyle = getStatusBadgeStyle(status);
  const displayText = getStatusDisplayText(status);
  const indicatorColor = getStatusIndicatorColor(status);

  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-1 rounded-full ${badgeStyle} ${className}`}
    >
      <span className={`w-2 h-2 ${indicatorColor} rounded-full mr-1`}></span>
      {showCount ? `${count} ${displayText}` : displayText}
    </span>
  );
};

export default StatusBadge;
