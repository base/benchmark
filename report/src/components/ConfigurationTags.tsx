import { formatValue, formatLabel } from "../utils/formatters";
import { camelToTitleCase } from "../utils/formatters";

interface ConfigurationTagsProps {
  testConfig: Record<string, unknown>;
  className?: string;
}

const ConfigurationTags = ({
  testConfig,
  className = "",
}: ConfigurationTagsProps) => {
  return (
    <div className={`flex flex-wrap gap-2 ${className}`}>
      {Object.entries(testConfig || {})
        .filter(([k]) => k !== "BenchmarkRun" && k !== "GasLimit")
        .map(([key, value]) => (
          <span
            key={key}
            title={`${camelToTitleCase(key)}: ${value}`}
            className="inline-flex items-center rounded-md bg-slate-50 px-2 py-1 text-xs text-slate-700 ring-1 ring-inset ring-slate-500/10"
          >
            <span className="mr-1.5 text-slate-500 font-normal">
              {camelToTitleCase(key)}:
            </span>
            {key === "GasLimit" ? (
              <span className="font-mono">
                {formatValue(Number(value), "gas")}
              </span>
            ) : (
              <span className="font-mono">
                {String(formatLabel(`${value}`))}
              </span>
            )}
          </span>
        ))}
    </div>
  );
};

export default ConfigurationTags;
