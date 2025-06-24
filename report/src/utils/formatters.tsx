import { ChartConfig } from "../types";

const PREFIXES = {
  "": 1,
  k: 1e3,
  M: 1e6,
  G: 1e9,
  T: 1e12,
  P: 1e15,
  E: 1e18,
  Z: 1e21,
  Y: 1e24,
};

const BINARY_PREFIXES = {
  "": 1,
  Ki: 1024,
  Mi: 1024 ** 2,
  Gi: 1024 ** 3,
  Ti: 1024 ** 4,
  Pi: 1024 ** 5,
  Ei: 1024 ** 6,
  Zi: 1024 ** 7,
  Yi: 1024 ** 8,
};

const TIME_UNITS = {
  ns: 1,
  us: 1e3, // Microsecond
  ms: 1e6, // Millisecond
  s: 1e9, // Second
};

const formatWithPrefix = (
  value: number,
  baseUnit: string,
  prefixes: { [key: string]: number },
  decimalPlaces: number = 1,
): string => {
  if (value === 0) return `0 ${baseUnit}`;

  const sortedPrefixes = Object.entries(prefixes).sort(
    ([, valA], [, valB]) => valB - valA,
  );

  for (const [prefix, multiplier] of sortedPrefixes) {
    if (Math.abs(value) >= multiplier) {
      return `${(value / multiplier).toFixed(decimalPlaces)} ${prefix}${baseUnit}`;
    }
  }
  // Should not happen if "" prefix with value 1 is included, but as fallback:
  return `${value.toFixed(decimalPlaces)} ${baseUnit}`;
};

export const formatLabel = (label: string) => {
  return label.length > 50 ? label.substring(0, 50) + "..." : label;
};

export const MetricValue = ({
  value,
  unit,
}: {
  value: number;
  unit: ChartConfig["unit"];
}) => {
  if (unit === undefined || typeof value !== "number" || isNaN(value)) {
    return value?.toString() ?? "";
  }

  return formatValue(value, unit);
};

export const formatValue = (
  value: number,
  unit?: ChartConfig["unit"],
): string => {
  if (unit === undefined || typeof value !== "number" || isNaN(value)) {
    return value?.toString() ?? "";
  }

  // Time Conversions (ns, us, ms, s)
  if (unit === "ns" || unit === "us" || unit === "ms" || unit === "s") {
    const baseValueInNs = value * (TIME_UNITS[unit] || 1); // Convert input to ns
    return formatWithPrefix(baseValueInNs, "s", {
      // Target unit is 's', prefixes based on ns
      n: TIME_UNITS.ns,
      µ: TIME_UNITS.us,
      m: TIME_UNITS.ms,
      "": TIME_UNITS.s, // Base unit 's' corresponding to 1e9 ns
    });
  }

  // Byte Conversions (bytes, KB, MB, GB) - using Binary Prefixes (KiB, MiB, GiB)
  if (unit === "bytes") {
    return formatWithPrefix(value, "B", BINARY_PREFIXES);
  }

  // Gas or Count (no scaling, use thousands separators)
  if (unit === "count") {
    return `${value.toLocaleString()}${unit !== "count" ? ` ${unit}` : ""}`;
  }

  // Gas per Second
  if (unit === "gas") {
    // Use SI prefixes for rate
    return formatWithPrefix(value, "gas", PREFIXES);
  }

  // Gas per Second
  if (unit === "gas/s") {
    // Use SI prefixes for rate
    return formatWithPrefix(value, "gas/s", PREFIXES);
  }

  // Default: just return the number as string
  return value.toString();
};

export const camelToTitleCase = (str: string) => {
  return str
    .replace(/([A-Z])/g, " $1")
    .replace(/[_-]+/g, " ")
    .split(" ")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
};
