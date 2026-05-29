import { formatValue, formatLabel } from "../utils/formatters";
import { camelToTitleCase } from "../utils/formatters";

interface ConfigurationTagsProps {
  testConfig: Record<string, unknown>;
  clientVersion?: string;
  className?: string;
}

const CONFIG_LABELS: Record<string, string> = {
  BlockTimeMilliseconds: "Block Time",
  ConsensusTimingMode: "Consensus Timing",
  GasLimit: "Gas Limit",
  NodeType: "Node Type",
  TargetGPS: "Target Gas/s",
  TransactionPayload: "Transaction Payload",
  ValidatorNodeType: "Validator Node Type",
};

const CONFIG_ORDER = [
  "TargetGPS",
  "GasLimit",
  "BlockTimeMilliseconds",
  "ConsensusTimingMode",
  "NodeType",
  "ValidatorNodeType",
  "TransactionPayload",
  "Roles",
];

const configLabel = (key: string): string =>
  CONFIG_LABELS[key] ?? camelToTitleCase(key);

const configValue = (key: string, value: unknown): string => {
  if (key === "GasLimit") {
    return formatValue(Number(value), "gas");
  }
  if (key === "TargetGPS") {
    return formatValue(Number(value), "gas/s");
  }
  if (key === "BlockTimeMilliseconds") {
    return formatValue(Number(value), "ms");
  }
  return String(formatLabel(`${value}`));
};

const configEntries = (testConfig: Record<string, unknown>) =>
  Object.entries(testConfig || {})
    .filter(([key, value]) => key !== "BenchmarkRun" && value !== "")
    .sort(([a], [b]) => {
      const aIndex = CONFIG_ORDER.indexOf(a);
      const bIndex = CONFIG_ORDER.indexOf(b);
      if (aIndex === -1 && bIndex === -1) {
        return a.localeCompare(b);
      }
      if (aIndex === -1) {
        return 1;
      }
      if (bIndex === -1) {
        return -1;
      }
      return aIndex - bIndex;
    });

const ConfigurationTags = ({
  testConfig,
  clientVersion,
  className = "",
}: ConfigurationTagsProps) => {
  return (
    <div className={`flex flex-wrap gap-2 ${className}`}>
      {clientVersion && (
        <span
          title={`Client Version: ${clientVersion}`}
          className="inline-flex items-center rounded-md bg-blue-50 px-2 py-1 text-xs text-blue-700 ring-1 ring-inset ring-blue-500/10"
        >
          <span className="mr-1.5 text-blue-500 font-normal">
            Client Version:
          </span>
          <span className="font-mono">{clientVersion}</span>
        </span>
      )}
      {configEntries(testConfig).map(([key, value]) => (
        <span
          key={key}
          title={`${configLabel(key)}: ${configValue(key, value)}`}
          className="inline-flex items-center rounded-md bg-slate-50 px-2 py-1 text-xs text-slate-700 ring-1 ring-inset ring-slate-500/10"
        >
          <span className="mr-1.5 text-slate-500 font-normal">
            {configLabel(key)}:
          </span>
          <span className="font-mono">{configValue(key, value)}</span>
        </span>
      ))}
    </div>
  );
};

export default ConfigurationTags;
