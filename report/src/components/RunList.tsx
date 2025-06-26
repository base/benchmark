import { groupBy } from "lodash";
import { useState } from "react";
import { formatValue, MetricValue } from "../utils/formatters";
import ThresholdDisplay from "../pages/ThresholdDisplay";
import { BenchmarkRunWithStatus } from "../types";
import StatusBadge from "./StatusBadge";
import StatusSummary from "./StatusSummary";
import ConfigurationTags from "./ConfigurationTags";
import Tooltip from "./Tooltip";
import clsx from "clsx";

interface ProvidedProps {
  groupedSections: {
    key: string;
    testName: string;
    runs: BenchmarkRunWithStatus[];
    diffKeyStart: number;
  }[];
  expandedSections: Set<string>;
  toggleSection: (section: string) => void;
}

type SortColumn =
  | "sendTxs"
  | "forkChoice"
  | "getPayload"
  | "gasPerSecond"
  | "newPayload";

type SortDirection = "asc" | "desc" | "disabled";

// Column definitions with tooltips
const COLUMN_DEFINITIONS = {
  sendTxs: {
    label: "Send Txs",
    tooltip: "Time taken to send transactions to the sequencer (seconds)",
    category: "sequencer" as const,
  },
  forkChoice: {
    label: "Fork Choice",
    tooltip: "Time for fork choice updates (seconds)",
    category: "sequencer" as const,
  },
  getPayload: {
    label: "Get Payload",
    tooltip: "Time to retrieve execution payload from sequencer (seconds)",
    category: "sequencer" as const,
  },
  gasPerSecond: {
    label: "Val. Gas/s",
    tooltip: "Gas processed per second by the validator",
    category: "validator" as const,
  },
  newPayload: {
    label: "New Payload",
    tooltip: "Time to process new payload on validator (seconds)",
    category: "validator" as const,
  },
} as const;

// Helper function to get metric value from run
const getMetricValue = (
  run: BenchmarkRunWithStatus,
  column: SortColumn,
): number => {
  switch (column) {
    case "sendTxs":
      return run.result?.sequencerMetrics?.sendTxs ?? 0;
    case "forkChoice":
      return run.result?.sequencerMetrics?.forkChoiceUpdated ?? 0;
    case "getPayload":
      return run.result?.sequencerMetrics?.getPayload ?? 0;
    case "gasPerSecond":
      return run.result?.validatorMetrics?.gasPerSecond ?? 0;
    case "newPayload":
      return run.result?.validatorMetrics?.newPayload ?? 0;
    default:
      return 0;
  }
};

const RunList = ({
  groupedSections,
  expandedSections,
  toggleSection,
}: ProvidedProps) => {
  const [sortColumn, setSortColumn] = useState<SortColumn>("gasPerSecond");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");

  const handleSort = (column: SortColumn) => {
    if (sortColumn === column) {
      // Cycle through: asc -> desc -> disabled -> asc
      if (sortDirection === "asc") {
        setSortDirection("desc");
      } else if (sortDirection === "desc") {
        setSortDirection("disabled");
      } else {
        setSortDirection("asc");
      }
    } else {
      setSortColumn(column);
      setSortDirection("asc");
    }
  };

  const getSortIcon = (column: SortColumn) => {
    if (sortColumn !== column || sortDirection === "disabled") return "↕";
    return sortDirection === "asc" ? "↑" : "↓";
  };

  const sortRuns = (runs: BenchmarkRunWithStatus[]) => {
    if (sortDirection === "disabled") return runs;

    return [...runs].sort((a, b) => {
      const aValue = getMetricValue(a, sortColumn);
      const bValue = getMetricValue(b, sortColumn);

      if (sortDirection === "asc") {
        return aValue - bValue;
      } else {
        return bValue - aValue;
      }
    });
  };

  const renderMetricCell = (
    run: BenchmarkRunWithStatus,
    column: SortColumn,
  ) => {
    const value = getMetricValue(run, column);

    // Handle threshold displays for specific columns
    if (column === "getPayload") {
      const warningThreshold =
        (run.thresholds?.warning?.["latency/get_payload"] ?? 0) / 1e9;
      const errorThreshold =
        (run.thresholds?.error?.["latency/get_payload"] ?? 0) / 1e9;

      return (
        <ThresholdDisplay
          value={value}
          warningThreshold={warningThreshold}
          errorThreshold={errorThreshold}
        >
          <MetricValue value={value} unit="s" />
        </ThresholdDisplay>
      );
    }

    if (column === "newPayload") {
      const warningThreshold =
        (run.thresholds?.warning?.["latency/new_payload"] ?? 0) / 1e9;
      const errorThreshold =
        (run.thresholds?.error?.["latency/new_payload"] ?? 0) / 1e9;

      return (
        <ThresholdDisplay
          value={value}
          warningThreshold={warningThreshold}
          errorThreshold={errorThreshold}
        >
          <MetricValue value={value} unit="s" />
        </ThresholdDisplay>
      );
    }

    // Default metric display
    const unit = column === "gasPerSecond" ? "gas/s" : "s";
    return <MetricValue value={value} unit={unit} />;
  };

  return (
    <div className="p-6 overflow-x-auto flex-grow border-slate-200">
      {groupedSections.map((section) => {
        const isExpanded = expandedSections.has(section.key);
        const statusCounts = groupBy(section.runs, "status");
        const sortedRuns = isExpanded ? sortRuns(section.runs) : section.runs;

        return (
          <div key={section.key} className="mb-4">
            <button
              className="flex items-center gap-4 w-full text-left p-2 rounded-lg transition-colors duration-150 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              onClick={() => toggleSection(section.key)}
            >
              <span className="inline-flex items-center justify-center w-6 h-6 text-slate-700">
                <svg
                  className={clsx(
                    "w-6 h-6 transition-transform duration-150",
                    isExpanded ? "" : "-rotate-90",
                  )}
                  fill="currentColor"
                >
                  <path d="m6 9 6 6 6-6" />
                </svg>
              </span>
              <div className="flex-1">
                <div className="flex items-center gap-4">
                  <span className="text-xl font-medium text-slate-900">
                    {formatValue(
                      Number(section.runs?.[0]?.testConfig?.GasLimit),
                      "gas/s",
                    )}
                  </span>
                  <StatusSummary statusCounts={statusCounts} />
                </div>
              </div>
            </button>

            {isExpanded && (
              <div className="mt-4">
                <table className="min-w-full">
                  <thead className="bg-slate-50">
                    <tr>
                      <td colSpan={2} />
                      <td
                        colSpan={3}
                        className="bg-blue-50 text-sm text-center py-3 font-medium text-blue-900 uppercase"
                      >
                        Sequencer Metrics
                      </td>
                      <td
                        colSpan={2}
                        className="bg-emerald-50 text-sm text-center py-3 font-medium text-emerald-900 uppercase"
                      >
                        Validator Metrics
                      </td>
                    </tr>
                    <tr>
                      <th className="px-6 py-3 text-left text-sm font-medium text-slate-700 tracking-wider uppercase">
                        Configuration
                      </th>
                      <th className="px-6 py-3 text-left text-sm font-medium text-slate-700 tracking-wider uppercase">
                        Status
                      </th>
                      {Object.entries(COLUMN_DEFINITIONS).map(([key, def]) => (
                        <th
                          key={key}
                          className={`px-6 py-3 text-left text-sm font-medium text-slate-700 tracking-wider cursor-pointer transition-colors duration-150 uppercase ${
                            def.category === "sequencer"
                              ? "bg-blue-50 hover:bg-blue-100"
                              : "bg-emerald-50 hover:bg-emerald-100"
                          }`}
                          onClick={() => handleSort(key as SortColumn)}
                        >
                          <Tooltip content={def.tooltip}>
                            <div className="flex items-center gap-1">
                              <span>{def.label.toUpperCase()}</span>
                              <span className="text-slate-400 font-normal">
                                {getSortIcon(key as SortColumn)}
                              </span>
                            </div>
                          </Tooltip>
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="bg-white">
                    {sortedRuns.map((run) => (
                      <tr
                        key={run.outputDir}
                        className="transition-colors duration-150"
                      >
                        <td className="px-4 py-2 text-sm text-slate-900">
                          <ConfigurationTags testConfig={run.testConfig} />
                        </td>
                        <td className="px-4 py-2 whitespace-nowrap text-sm">
                          <StatusBadge
                            status={run.status}
                            className="text-xs"
                          />
                        </td>
                        {Object.keys(COLUMN_DEFINITIONS).map((column) => (
                          <td
                            key={column}
                            className="px-4 py-2 whitespace-nowrap text-sm text-slate-600 font-mono"
                          >
                            {renderMetricCell(run, column as SortColumn)}
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};

export default RunList;
