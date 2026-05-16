import { Link, useParams } from "react-router-dom";
import { useMemo } from "react";
import Navbar from "../components/Navbar";
import StatCard, { Stat, StatGrid } from "../components/StatCard";
import PercentileBarChart, {
  PercentileBarRow,
} from "../components/PercentileBar";
import ThroughputChart from "../components/ThroughputChart";
import ConfigCard from "../components/ConfigCard";
import { useLoadTestResult } from "../utils/useDataSeries";
import {
  durationToNanos,
  formatDuration,
  formatEthFromWei,
  formatGasVerbose,
  formatGpsVerbose,
  formatLoadTestTimestamp,
  formatPercent,
  formatTps,
} from "../utils/formatters";
import {
  FlashblocksLatencyStats,
  LatencyStats,
  LoadTestResult,
} from "../types";

const buildLatencyRows = (
  stats: LatencyStats | FlashblocksLatencyStats,
): PercentileBarRow[] => {
  const rows: PercentileBarRow[] = [
    {
      label: "min",
      numericValue: durationToNanos(stats.min),
      display: formatDuration(stats.min),
    },
    {
      label: "p50",
      numericValue: durationToNanos(stats.p50),
      display: formatDuration(stats.p50),
    },
    {
      label: "mean",
      numericValue: durationToNanos(stats.mean),
      display: formatDuration(stats.mean),
    },
  ];

  if ("p90" in stats && stats.p90) {
    rows.push({
      label: "p90",
      numericValue: durationToNanos(stats.p90),
      display: formatDuration(stats.p90),
    });
  }

  rows.push(
    {
      label: "p95",
      numericValue: durationToNanos(stats.p95),
      display: formatDuration(stats.p95),
    },
    {
      label: "p99",
      numericValue: durationToNanos(stats.p99),
      display: formatDuration(stats.p99),
      emphasized: true,
    },
    {
      label: "max",
      numericValue: durationToNanos(stats.max),
      display: formatDuration(stats.max),
    },
  );

  return rows;
};

const SwapsPerSecondHero = ({ tps }: { tps: number }) => (
  <section className="rounded-lg bg-white border border-slate-200 px-8 py-10 flex flex-col items-center text-center">
    <div className="text-7xl font-semibold text-slate-900 tabular-nums tracking-tight">
      {tps.toLocaleString(undefined, {
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
      })}
    </div>
    <div className="mt-2 text-base text-slate-500">Swaps/s</div>
  </section>
);

const SummarySection = ({ result }: { result: LoadTestResult }) => {
  const submitted = result.throughput.total_submitted;
  const confirmed = result.throughput.total_confirmed;
  const failed = result.throughput.total_failed;
  const blockRange = result.block_range;
  const hasConfirmedBlockRange =
    typeof blockRange?.first_block === "number" &&
    typeof blockRange.last_block === "number";

  return (
    <StatCard title="Summary">
      <StatGrid>
        <Stat
          label="Duration"
          value={formatDuration(result.throughput.duration)}
        />
        <Stat label="Submitted" value={submitted.toLocaleString()} />
        <Stat
          label="Confirmed"
          value={confirmed.toLocaleString()}
          hint={formatPercent(confirmed, submitted) + " of submitted"}
        />
        <Stat label="Failed" value={failed.toLocaleString()} />
        <Stat label="Avg TPS" value={formatTps(result.throughput.tps)} />
        <Stat
          label="Avg gas/s"
          value={formatGpsVerbose(result.throughput.gps)}
        />
        <Stat
          label="Total gas"
          value={formatGasVerbose(result.gas.total_gas)}
          hint={`${result.gas.avg_gas.toLocaleString()} avg / tx`}
        />
        <Stat
          label="Total cost"
          value={formatEthFromWei(result.gas.total_cost_wei)}
          hint={`${result.gas.avg_gas_price.toLocaleString()} wei avg gas price`}
        />
        {blockRange && (
          <Stat
            label="Block range"
            value={
              hasConfirmedBlockRange
                ? `${blockRange.first_block.toLocaleString()} → ${blockRange.last_block.toLocaleString()}`
                : "No confirmed transactions"
            }
            hint={`${blockRange.block_count.toLocaleString()} blocks`}
          />
        )}
      </StatGrid>
    </StatCard>
  );
};

const LoadTestDetail = () => {
  const { network, timestamp } = useParams();
  const {
    data: result,
    isLoading,
    error,
  } = useLoadTestResult(network, timestamp);

  const blockLatencyRows = useMemo(
    () => (result ? buildLatencyRows(result.block_latency) : []),
    [result],
  );
  const flashblocksLatencyRows = useMemo(
    () => (result ? buildLatencyRows(result.flashblocks_latency) : []),
    [result],
  );

  return (
    <div className="min-h-screen bg-slate-50">
      <Navbar />
      <main className="px-8 py-6 max-w-5xl mx-auto flex flex-col gap-y-6">
        <header className="flex items-center justify-between gap-x-4">
          <div>
            <Link
              to={`/load-tests/${network ?? "sepolia"}/all`}
              className="text-sm text-blue-600 hover:underline"
            >
              View all runs →
            </Link>
            <h1 className="text-2xl font-semibold text-slate-900 mt-2">
              {timestamp ? formatLoadTestTimestamp(timestamp) : "Load test"}
            </h1>
            <p className="text-sm text-slate-500 mt-1">
              Network: <span className="font-mono">{network}</span>
              {timestamp && (
                <>
                  {" · "}
                  <span className="font-mono text-slate-400">{timestamp}</span>
                </>
              )}
            </p>
          </div>
        </header>

        {isLoading && (
          <div className="text-sm text-slate-500">Loading load test…</div>
        )}

        {error && (
          <div className="border border-red-200 bg-red-50 text-red-800 rounded-lg p-4 text-sm">
            Failed to load load test result: {String(error)}
          </div>
        )}

        {result && (
          <>
            <SwapsPerSecondHero tps={result.throughput.tps} />

            {result.throughput_timeseries &&
              result.throughput_timeseries.length > 1 && (
                <StatCard title="Throughput over time">
                  <ThroughputChart
                    samples={result.throughput_timeseries}
                    avgTps={result.throughput.tps}
                    avgGps={result.throughput.gps}
                  />
                </StatCard>
              )}

            {result.config && <ConfigCard config={result.config} />}

            <SummarySection result={result} />

            <StatCard title="Block latency (submit → block)">
              <PercentileBarChart
                rows={blockLatencyRows}
                barColorClass="bg-amber-500"
              />
            </StatCard>

            <StatCard
              title={`Flashblocks latency (submit → flashblock) · ${result.flashblocks_latency.count.toLocaleString()} samples`}
            >
              <PercentileBarChart
                rows={flashblocksLatencyRows}
                barColorClass="bg-fuchsia-500"
              />
            </StatCard>

            <StatCard title="Top failure reasons">
              {result.top_failure_reasons.length === 0 ? (
                <div className="text-sm text-slate-500">
                  No failures recorded.
                </div>
              ) : (
                <ul className="text-sm text-slate-700 divide-y divide-slate-100">
                  {result.top_failure_reasons.map(([reason, count]) => (
                    <li
                      key={reason}
                      className="py-2 flex justify-between gap-x-4"
                    >
                      <span>{reason}</span>
                      <span className="font-mono">
                        {count.toLocaleString()}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </StatCard>
          </>
        )}
      </main>
    </div>
  );
};

export default LoadTestDetail;
