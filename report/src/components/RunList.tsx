import { groupBy } from "lodash";
import { useState } from "react";
import { formatValue, MetricValue } from "../utils/formatters";
import { camelToTitleCase } from "../utils/formatters";
import { formatLabel } from "../utils/formatters";
import ThresholdDisplay from "../pages/ThresholdDisplay";
import { BenchmarkRunWithStatus } from "../types";

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
      let aValue: number;
      let bValue: number;

      switch (sortColumn) {
        case "sendTxs":
          aValue = a.result?.sequencerMetrics?.sendTxs ?? 0;
          bValue = b.result?.sequencerMetrics?.sendTxs ?? 0;
          break;
        case "forkChoice":
          aValue = a.result?.sequencerMetrics?.forkChoiceUpdated ?? 0;
          bValue = b.result?.sequencerMetrics?.forkChoiceUpdated ?? 0;
          break;
        case "getPayload":
          aValue = a.result?.sequencerMetrics?.getPayload ?? 0;
          bValue = b.result?.sequencerMetrics?.getPayload ?? 0;
          break;
        case "gasPerSecond":
          aValue = a.result?.validatorMetrics?.gasPerSecond ?? 0;
          bValue = b.result?.validatorMetrics?.gasPerSecond ?? 0;
          break;
        case "newPayload":
          aValue = a.result?.validatorMetrics?.newPayload ?? 0;
          bValue = b.result?.validatorMetrics?.newPayload ?? 0;
          break;
        default:
          return 0;
      }

      if (sortDirection === "asc") {
        return aValue - bValue;
      } else {
        return bValue - aValue;
      }
    });
  };

  return (
    <div className="p-8 overflow-x-auto border-t border-slate-200">
      {/* Render all grouped sections */}
      {groupedSections.map((section) => {
        const isExpanded = expandedSections.has(section.key);

        // Compute pass/fail counts
        const { warning, error, success, incomplete, fatal } = groupBy(
          section.runs,
          "status",
        );

        // Sort runs if section is expanded
        const sortedRuns = isExpanded ? sortRuns(section.runs) : section.runs;

        return (
          <div key={section.key} className="mb-10">
            {/* Clickable heading */}
            <button
              className="flex items-center gap-4 text-lg font-semibold mb-2 focus:outline-none"
              onClick={() => {
                toggleSection(section.key);
              }}
            >
              <span className="inline-block w-5 text-center">
                {isExpanded ? "▼" : "►"}
              </span>
              <span>
                {formatValue(
                  Number(section.runs?.[0]?.testConfig?.GasLimit),
                  "gas/s",
                )}
              </span>
              {/* Pass/Fail summary */}
              <span className="text-base font-normal text-slate-500">
                <span className="text-green-600">
                  {success?.length ?? 0} Passed
                </span>
                {fatal?.length > 0 && (
                  <>
                    {" / "}
                    <span className="text-red-600">{fatal.length} Errored</span>
                  </>
                )}

                {error?.length > 0 && (
                  <>
                    {" / "}
                    <span className="text-red-600">{error.length} Failed</span>
                  </>
                )}
                {warning?.length > 0 && (
                  <>
                    {" / "}
                    <span className="text-yellow-600">
                      {warning.length} Warning
                    </span>
                  </>
                )}
                {incomplete?.length > 0 && (
                  <>
                    {" / "}
                    <span className="text-yellow-600">
                      {incomplete.length} In Progress
                    </span>
                  </>
                )}
              </span>
            </button>
            {/* Only render table if expanded */}
            {isExpanded && (
              <table className="min-w-full divide-y divide-slate-200 rounded-lg mb-8">
                <thead>
                  <tr>
                    <td colSpan={3} />
                    <td
                      colSpan={3}
                      className="bg-blue-100 text-sm text-center py-2 font-bold"
                    >
                      Sequencer
                    </td>
                    <td
                      colSpan={2}
                      className="bg-green-100 text-sm text-center py-2 font-bold"
                    >
                      Validator
                    </td>
                  </tr>
                  <tr>
                    <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Test Name
                    </th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Config
                    </th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Status
                    </th>
                    <th
                      className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider bg-blue-100 cursor-pointer hover:bg-blue-200"
                      onClick={() => handleSort("sendTxs")}
                    >
                      Send Txs {getSortIcon("sendTxs")}
                    </th>
                    <th
                      className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider bg-blue-100 cursor-pointer hover:bg-blue-200"
                      onClick={() => handleSort("forkChoice")}
                    >
                      Fork Choice {getSortIcon("forkChoice")}
                    </th>
                    <th
                      className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider bg-blue-100 cursor-pointer hover:bg-blue-200"
                      onClick={() => handleSort("getPayload")}
                    >
                      Get Payload {getSortIcon("getPayload")}
                    </th>
                    <th
                      className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider bg-green-100 cursor-pointer hover:bg-green-200"
                      onClick={() => handleSort("gasPerSecond")}
                    >
                      Val. Gas/s {getSortIcon("gasPerSecond")}
                    </th>
                    <th
                      className="px-4 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider bg-green-100 cursor-pointer hover:bg-green-200"
                      onClick={() => handleSort("newPayload")}
                    >
                      New Payload {getSortIcon("newPayload")}
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-slate-200 border-left border border-slate-200">
                  {sortedRuns.map((run) => {
                    const newPayloadWarningThreshold =
                      run.thresholds?.warning?.["latency/new_payload"] ?? 0;
                    const newPayloadErrorThreshold =
                      run.thresholds?.error?.["latency/new_payload"] ?? 0;
                    const getPayloadWarningThreshold =
                      run.thresholds?.warning?.["latency/get_payload"] ?? 0;
                    const getPayloadErrorThreshold =
                      run.thresholds?.error?.["latency/get_payload"] ?? 0;

                    return (
                      <tr key={run.outputDir} className="hover:bg-slate-50">
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-900 align-top">
                          {run.testName}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-900 align-top">
                          <div className="mt-1 flex flex-wrap gap-1">
                            {Object.entries(run.testConfig || {}).map(
                              ([key, value]) => (
                                <span
                                  key={key}
                                  title={`${camelToTitleCase(key)}: ${value}`}
                                  className="inline-flex items-center rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-600 ring-1 ring-inset ring-slate-500/10 overflow-hidden text-ellipsis whitespace-nowrap"
                                >
                                  <span className="mr-1 text-slate-500">
                                    {camelToTitleCase(key)}:
                                  </span>
                                  {key === "GasLimit" ? (
                                    <pre>
                                      {formatValue(Number(value), "gas")}
                                    </pre>
                                  ) : (
                                    String(formatLabel(`${value}`))
                                  )}
                                </span>
                              ),
                            )}
                          </div>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                          {run.status === "incomplete" ? (
                            <span className="inline-flex items-center rounded-md bg-yellow-50 px-2 py-1 text-xs font-medium text-yellow-700 ring-1 ring-inset ring-yellow-600/20">
                              In Progress
                            </span>
                          ) : run.status === "success" ? (
                            <span className="inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700 ring-1 ring-inset ring-green-600/20">
                              Success
                            </span>
                          ) : run.status === "warning" ? (
                            <span className="inline-flex items-center rounded-md bg-yellow-50 px-2 py-1 text-xs font-medium text-yellow-700 ring-1 ring-inset ring-yellow-600/20">
                              Warning
                            </span>
                          ) : run.status === "error" ? (
                            <span className="inline-flex items-center rounded-md bg-red-50 px-2 py-1 text-xs font-medium text-red-700 ring-1 ring-inset ring-red-600/20">
                              Error
                            </span>
                          ) : (
                            <span className="inline-flex items-center rounded-md bg-gray-50 px-2 py-1 text-xs font-medium text-gray-700 ring-1 ring-inset ring-gray-600/20">
                              {run.status}
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                          <MetricValue
                            value={run.result?.sequencerMetrics?.sendTxs ?? 0}
                            unit="s"
                          />
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                          <MetricValue
                            value={
                              run.result?.sequencerMetrics?.forkChoiceUpdated ??
                              0
                            }
                            unit="s"
                          />
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                          <ThresholdDisplay
                            value={
                              run.result?.sequencerMetrics?.getPayload ?? 0
                            }
                            warningThreshold={getPayloadWarningThreshold / 1e9}
                            errorThreshold={getPayloadErrorThreshold / 1e9}
                          >
                            <MetricValue
                              value={
                                run.result?.sequencerMetrics?.getPayload ?? 0
                              }
                              unit="s"
                            />
                          </ThresholdDisplay>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                          <MetricValue
                            value={
                              run.result?.validatorMetrics?.gasPerSecond ?? 0
                            }
                            unit="gas/s"
                          />
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500">
                          <ThresholdDisplay
                            value={
                              run.result?.validatorMetrics?.newPayload ?? 0
                            }
                            warningThreshold={newPayloadWarningThreshold / 1e9}
                            errorThreshold={newPayloadErrorThreshold / 1e9}
                          >
                            <MetricValue
                              value={
                                run.result?.validatorMetrics?.newPayload ?? 0
                              }
                              unit="s"
                            />
                          </ThresholdDisplay>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </div>
        );
      })}
    </div>
  );
};

export default RunList;
