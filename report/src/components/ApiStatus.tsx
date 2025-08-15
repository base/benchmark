import { useApiHealth } from "../utils/useDataSeries";

interface ApiStatusProps {
  className?: string;
}

/**
 * Component to display API connection status
 * Shows a small indicator in the navbar or footer
 */
export const ApiStatus = ({ className = "" }: ApiStatusProps) => {
  const { data: healthData, error, isLoading } = useApiHealth();

  // Don't show anything while loading initially
  if (isLoading && !healthData) {
    return null;
  }

  const getStatusInfo = () => {
    if (error) {
      return {
        status: "error",
        color: "bg-red-500",
        title: "API Connection Error",
        message: "Unable to connect to benchmark data service",
      };
    }

    if (healthData?.status === "healthy") {
      return {
        status: "healthy",
        color: "bg-emerald-500",
        title: "API Connected",
        message: "Connected to benchmark data service",
      };
    }

    return {
      status: "unknown",
      color: "bg-yellow-500",
      title: "API Status Unknown",
      message: "Benchmark data service status unknown",
    };
  };

  const statusInfo = getStatusInfo();

  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <div
        className={`w-2 h-2 rounded-full ${statusInfo.color}`}
        title={statusInfo.title}
      />
      <span className="text-xs text-slate-600 hidden sm:inline">
        {statusInfo.message}
      </span>
    </div>
  );
};

export default ApiStatus;
