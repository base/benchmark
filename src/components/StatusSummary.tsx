import StatusBadge from "./StatusBadge";

interface StatusSummaryProps {
  statusCounts: {
    success?: unknown[];
    fatal?: unknown[];
    error?: unknown[];
    warning?: unknown[];
    incomplete?: unknown[];
  };
  className?: string;
}

const StatusSummary = ({
  statusCounts,
  className = "",
}: StatusSummaryProps) => {
  const { success, fatal, error, warning, incomplete } = statusCounts;

  return (
    <div className={`flex items-center gap-3 text-sm ${className}`}>
      {success && success.length > 0 && (
        <StatusBadge status="success" showCount count={success.length} />
      )}
      {fatal && fatal.length > 0 && (
        <StatusBadge status="error" showCount count={fatal.length} />
      )}
      {error && error.length > 0 && (
        <StatusBadge status="error" showCount count={error.length} />
      )}
      {warning && warning.length > 0 && (
        <StatusBadge status="warning" showCount count={warning.length} />
      )}
      {incomplete && incomplete.length > 0 && (
        <StatusBadge status="incomplete" showCount count={incomplete.length} />
      )}
    </div>
  );
};

export default StatusSummary;
