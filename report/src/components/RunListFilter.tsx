import { Link } from "react-router-dom";
import { camelToTitleCase, formatLabel } from "../utils/formatters";
import { FilterValue } from "../filter";

interface ProvidedProps {
  filterOptions: Record<string, FilterValue[]>;
  filterSelections: Record<string, FilterValue>;
  updateFilterSelection: (key: string, value: string | null) => void;
}

const RunListFilter = ({
  filterOptions,
  filterSelections,
  updateFilterSelection,
}: ProvidedProps) => {
  return (
    <div className="flex justify-between items-center mb-4 w-full">
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
                <select
                  className="bg-white border border-slate-300 rounded-md px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
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
                </select>
              </div>
            );
          })}
      </div>
      <Link to="/run-comparison">
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
  );
};

export default RunListFilter;
