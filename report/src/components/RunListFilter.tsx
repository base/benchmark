import { Link } from "react-router-dom";
import { useState } from "react";
import { camelToTitleCase, formatLabel } from "../utils/formatters";
import { FilterValue } from "../filter";
import { BenchmarkRunWithStatus } from "../types";
import { downloadAllRuns, downloadAllRunsCSV } from "../utils/downloadUtils";
import Select from "./Select";
import FloatingDropdown from "./FloatingDropdown";
import clsx from "clsx";

interface ProvidedProps {
  benchmarkRunId: string;
  filterOptions: Record<string, FilterValue[]>;
  filterSelections: Record<string, FilterValue>;
  updateFilterSelection: (key: string, value: string | null) => void;
  allRuns: BenchmarkRunWithStatus[];
  testName: string;
}

const RunListFilter = ({
  benchmarkRunId,
  filterOptions,
  filterSelections,
  updateFilterSelection,
  allRuns,
  testName,
}: ProvidedProps) => {
  const [isDownloadOpen, setIsDownloadOpen] = useState(false);

  const handleDownloadAllJSON = () => {
    downloadAllRuns(allRuns, testName);
    setIsDownloadOpen(false);
  };

  const handleDownloadAllCSV = () => {
    downloadAllRunsCSV(allRuns, testName);
    setIsDownloadOpen(false);
  };

  const downloadTrigger = (
    <button
      className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors flex items-center gap-2 flex-shrink-0 ml-4"
      aria-label="Download all run information"
    >
      <svg
        className="w-4 h-4"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
        />
      </svg>
      Download All ({allRuns.length})
      <svg
        className={clsx(
          "w-4 h-4 transition-transform duration-150",
          isDownloadOpen && "rotate-180",
        )}
        fill="currentColor"
        viewBox="0 0 20 20"
      >
        <path
          fillRule="evenodd"
          d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
          clipRule="evenodd"
        />
      </svg>
    </button>
  );

  const downloadContent = (
    <>
      <button
        onClick={handleDownloadAllJSON}
        className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
      >
        <svg
          className="w-4 h-4"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
          />
        </svg>
        Download All as JSON
      </button>
      <button
        onClick={handleDownloadAllCSV}
        className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
      >
        <svg
          className="w-4 h-4"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
          />
        </svg>
        Download All as CSV
      </button>
    </>
  );

  return (
    <div className="flex justify-between items-start mb-4 w-full">
      <div className="flex flex-wrap gap-4">
        {Object.entries(filterOptions)
          .sort((a, b) => a[0].localeCompare(b[0]))
          .map(([key, availableValues]) => {
            const currentValue = filterSelections[key] ?? "any";
            return (
              <div key={key}>
                <div className="text-sm text-slate-500 mb-1">
                  {camelToTitleCase(key)}
                </div>
                <Select
                  value={String(currentValue)}
                  onChange={(e) => {
                    const newValue = e.target.value;
                    updateFilterSelection(
                      key,
                      newValue === "any" ? null : newValue,
                    );
                  }}
                >
                  <option value="any">Any</option>
                  {availableValues.map((val) => (
                    <option value={String(val)} key={String(val)}>
                      {formatLabel(String(val))}
                    </option>
                  ))}
                </Select>
              </div>
            );
          })}
      </div>

      <div className="flex items-center gap-3">
        {/* Download All Button */}
        <FloatingDropdown
          trigger={downloadTrigger}
          isOpen={isDownloadOpen}
          onToggle={() => setIsDownloadOpen(!isDownloadOpen)}
          onClose={() => setIsDownloadOpen(false)}
          placement="bottom-right"
        >
          {downloadContent}
        </FloatingDropdown>

        {/* View Block Metrics Button */}
        <Link to={`/run-comparison/${benchmarkRunId}`}>
          <button
            type="button"
            className="px-4 py-2 bg-slate-100 text-slate-900 rounded hover:bg-slate-200 transition-colors flex items-center gap-2"
          >
            <span role="img" aria-label="Blocks">
              📊
            </span>{" "}
            View Block Metrics
          </button>
        </Link>
      </div>
    </div>
  );
};

export default RunListFilter;
