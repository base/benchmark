import { ChartConfig } from "../types";

const TIME_UNITS = {
  ns: 1,
  us: 1000,
  ms: 1000 * 1000,
  s: 1000 * 1000 * 1000,
};

const BYTE_UNITS = {
  bytes: 1,
  KB: 1024,
  MB: 1024 * 1024,
  GB: 1024 * 1024 * 1024,
};

export const formatValue = (
  value: number,
  unit?: ChartConfig["unit"],
): string => {
  if (!unit || typeof value !== "number") {
    return value?.toString() ?? "";
  }

  // Time Conversions (ns, ms, s)
  if (unit === "ns" || unit === "ms" || unit === "s") {
    const baseValue =
      unit === "ns"
        ? value
        : unit === "ms"
          ? value * TIME_UNITS.ms
          : value * TIME_UNITS.s; // Convert input to ns

    if (baseValue >= TIME_UNITS.s)
      return `${(baseValue / TIME_UNITS.s).toFixed(1)} s`;
    if (baseValue >= TIME_UNITS.ms)
      return `${(baseValue / TIME_UNITS.ms).toFixed(1)} ms`;
    if (baseValue >= TIME_UNITS.us)
      return `${(baseValue / TIME_UNITS.us).toFixed(1)} µs`; // Add microseconds
    return `${baseValue.toFixed(0)} ns`;
  }

  // Byte Conversions (bytes, KB, MB, GB)
  if (unit === "bytes") {
    if (value >= BYTE_UNITS.GB)
      return `${(value / BYTE_UNITS.GB).toFixed(1)} GB`;
    if (value >= BYTE_UNITS.MB)
      return `${(value / BYTE_UNITS.MB).toFixed(1)} MB`;
    if (value >= BYTE_UNITS.KB)
      return `${(value / BYTE_UNITS.KB).toFixed(1)} KB`;
    return `${value.toFixed(0)} bytes`;
  }

  // Gas or Count (no scaling)
  if (unit === "gas" || unit === "count") {
    // Add thousands separators, add unit if not count
    return `${value.toLocaleString()}${unit !== "count" ? ` ${unit}` : ""}`;
  }

  // Default: just return the number as string
  return value.toString();
};
