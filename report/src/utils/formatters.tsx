import { ChartConfig, RustDuration } from "../types";

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

export const durationToNanos = (d: RustDuration): number =>
  d.secs * 1e9 + d.nanos;

export const durationToMs = (d: RustDuration): number =>
  d.secs * 1000 + d.nanos / 1e6;

export const formatDuration = (d: RustDuration): string =>
  formatValue(durationToNanos(d), "ns");

export const formatTps = (n: number): string => `${n.toFixed(1)} tx/s`;

export const formatGps = (n: number): string => formatValue(n, "gas/s");

const WRITTEN_PREFIXES: Record<string, string> = {
  "": "",
  k: "thousand ",
  M: "million ",
  G: "billion ",
  T: "trillion ",
};

const formatGasWritten = (value: number, suffix: string): string => {
  if (value === 0) return `0 gas${suffix}`;
  const sortedPrefixes = Object.entries(PREFIXES).sort(([, a], [, b]) => b - a);
  for (const [prefix, multiplier] of sortedPrefixes) {
    if (Math.abs(value) >= multiplier) {
      const written = WRITTEN_PREFIXES[prefix] ?? `${prefix} `;
      return `${(value / multiplier).toFixed(1)} ${written}gas${suffix}`;
    }
  }
  return `${value.toFixed(1)} gas${suffix}`;
};

export const formatGasVerbose = (n: number): string => formatGasWritten(n, "");

export const formatGpsVerbose = (n: number): string =>
  formatGasWritten(n, "/s");

export const formatPercent = (
  numerator: number,
  denominator: number,
): string =>
  denominator === 0 ? "—" : `${((numerator / denominator) * 100).toFixed(2)}%`;

export const formatEthFromWei = (wei: number): string => {
  // total_cost_wei may exceed Number.MAX_SAFE_INTEGER; once the producer
  // stringifies it (upgrades.md P0 #1) accept `string` here and parse via BigInt.
  // Today we accept the precision loss because the display only needs 6 decimals.
  return `${(wei / 1e18).toFixed(6)} ETH`;
};

export const formatEthFromWeiString = (wei: string): string => {
  // funding_amount and swap_token_amount come over the wire as decimal strings
  // (Rust u128 → JSON string) so they don't lose precision in JSON. Use BigInt
  // for the integer-ETH part; only the fractional remainder needs Number math
  // (which is safe because it's < 1 ETH worth of wei).
  try {
    const w = BigInt(wei);
    const oneEth = 10n ** 18n;
    const whole = w / oneEth;
    const remainder = w % oneEth;
    if (remainder === 0n) return `${whole.toString()} ETH`;
    const frac = (Number(remainder) / 1e18).toFixed(6).slice(2);
    return `${whole.toString()}.${frac} ETH`;
  } catch {
    return `${wei} wei`;
  }
};

const LOAD_TEST_TIMESTAMP_RE =
  /^(\d{4})-(\d{2})-(\d{2})-(\d{2})-(\d{2})-(\d{2})$/;

export const parseLoadTestTimestamp = (raw: string): Date | null => {
  // Format produced by base-load-test: "YYYY-MM-DD-HH-MM-SS" (UTC, no zone in
  // the string but the producer writes UTC). Returning null on parse failure
  // lets callers degrade to showing the raw string instead of crashing.
  const m = LOAD_TEST_TIMESTAMP_RE.exec(raw);
  if (!m) return null;
  const [, y, mo, d, h, mi, s] = m;
  const ts = Date.UTC(
    Number(y),
    Number(mo) - 1,
    Number(d),
    Number(h),
    Number(mi),
    Number(s),
  );
  return Number.isNaN(ts) ? null : new Date(ts);
};

export const formatLoadTestTimestamp = (raw: string): string => {
  const d = parseLoadTestTimestamp(raw);
  if (!d) return raw;
  return Intl.DateTimeFormat("en-US", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "UTC",
  }).format(d);
};

export const camelToTitleCase = (str: string) => {
  return str
    .replace(/([A-Z])/g, " $1")
    .replace(/[_-]+/g, " ")
    .split(" ")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
};
