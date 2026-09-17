import { Link, useParams } from "react-router-dom";
import { type ReactNode, useMemo } from "react";
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
  formatGpsVerbose,
  formatLoadTestTimestamp,
  formatTps,
} from "../utils/formatters";
import {
  BlockRange,
  FlashblocksLatencyStats,
  LatencyStats,
  LoadTestResult,
} from "../types";

const formatBlockRange = (range: BlockRange): string => {
  if (
    typeof range.first_block === "number" &&
    typeof range.last_block === "number"
  ) {
    return `${range.first_block.toLocaleString()} → ${range.last_block.toLocaleString()}`;
  }
  return "No confirmed transactions";
};

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

const SwapsPerSecondHero = ({ tps, label }: { tps: number; label: string }) => (
  <section className="rounded-lg bg-white border border-slate-200 px-8 py-10 flex flex-col items-center text-center">
    <div className="text-7xl font-semibold text-slate-900 tabular-nums tracking-tight">
      {tps.toLocaleString(undefined, {
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
      })}
    </div>
    <div className="mt-2 text-base text-slate-500">{label}</div>
  </section>
);

const RESULTS_TOOLTIP = (
  <p className="max-w-xs leading-snug">
    Client-to-client end-to-end latency: from the moment a transaction is
    submitted via <code>eth_sendRawTransaction</code> to when the client
    receives and processes it (seeing its own txHash in a block). This mirrors
    what you witness watching the chain live: sustained TPS over a period of
    time. Use this against OKRs like &ldquo;hit 3k swaps/s on the chain.&rdquo;
  </p>
);

const ResultsSummary = ({ result }: { result: LoadTestResult }) => {
  const { throughput, block_range: blockRange } = result;

  return (
    <StatCard title="Results" titleTooltip={RESULTS_TOOLTIP}>
      <StatGrid>
        <Stat
          label="Submitted"
          value={throughput.total_submitted.toLocaleString()}
        />
        <Stat
          label="Confirmed"
          value={throughput.total_confirmed.toLocaleString()}
        />
        <Stat label="TPS" value={formatTps(throughput.tps)} />
        <Stat label="Gas/s" value={formatGpsVerbose(throughput.gps)} />
        {blockRange && (
          <Stat
            label="Block range"
            value={formatBlockRange(blockRange)}
            hint={`${blockRange.block_count.toLocaleString()} blocks`}
          />
        )}
      </StatGrid>
    </StatCard>
  );
};

interface LoadTestReportContentProps {
  result: LoadTestResult;
  title: string;
  subtitle: ReactNode;
  backLink?: {
    to: string;
    label: string;
  };
  network?: string;
}

export const LoadTestReportContent = ({
  result,
  title,
  subtitle,
  backLink,
  network,
}: LoadTestReportContentProps) => {
  const headlineTps = result.throughput.tps;
  const headlineFlashblocksLatency = result.flashblocks_latency;

  const headlineBlockLatencyRows = useMemo(
    () => buildLatencyRows(result.block_latency),
    [result.block_latency],
  );
  const headlineFlashblocksRows = useMemo(
    () => buildLatencyRows(headlineFlashblocksLatency),
    [headlineFlashblocksLatency],
  );

  // Proofs runs are not swap-based, so the hero reads as raw TPS instead of Swaps/s.
  const headlineLabel = network === "proofs" ? "TPS" : "Swaps/s";

  return (
    <>
      <header className="flex items-center justify-between gap-x-4">
        <div>
          {backLink && (
            <Link
              to={backLink.to}
              className="text-sm text-blue-600 hover:underline"
            >
              {backLink.label}
            </Link>
          )}
          <h1 className="text-2xl font-semibold text-slate-900 mt-2">
            {title}
          </h1>
          <p className="text-sm text-slate-500 mt-1">{subtitle}</p>
        </div>
      </header>

      <SwapsPerSecondHero tps={headlineTps} label={headlineLabel} />

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

      <ResultsSummary result={result} />

      <StatCard title="Block latency (e2e client observed)">
        <PercentileBarChart
          rows={headlineBlockLatencyRows}
          barColorClass="bg-amber-500"
        />
      </StatCard>

      <StatCard
        title={`Flashblocks latency (e2e client observed) · ${headlineFlashblocksLatency.count.toLocaleString()} samples`}
      >
        <PercentileBarChart
          rows={headlineFlashblocksRows}
          barColorClass="bg-fuchsia-500"
        />
      </StatCard>

      <StatCard title="Top failure reasons">
        {(() => {
          const reverted = result.throughput.total_reverted;
          const reasons: [string, number][] = [
            ...(reverted > 0
              ? [["reverted", reverted] as [string, number]]
              : []),
            ...result.top_failure_reasons,
          ];
          if (reasons.length === 0) {
            return (
              <div className="text-sm text-slate-500">
                No failures recorded.
              </div>
            );
          }
          return (
            <ul className="text-sm text-slate-700 divide-y divide-slate-100">
              {reasons.map(([reason, count]) => (
                <li key={reason} className="py-2 flex justify-between gap-x-4">
                  <span>{reason}</span>
                  <span className="font-mono">{count.toLocaleString()}</span>
                </li>
              ))}
            </ul>
          );
        })()}
      </StatCard>
    </>
  );
};

const LoadTestDetail = () => {
  const { network, timestamp } = useParams();
  const {
    data: result,
    isLoading,
    error,
  } = useLoadTestResult(network, timestamp);

  return (
    <div className="min-h-screen bg-slate-50">
      <Navbar />
      <main className="px-8 py-6 max-w-5xl mx-auto flex flex-col gap-y-6">
        {isLoading && (
          <div className="text-sm text-slate-500">Loading load test…</div>
        )}

        {error && (
          <div className="border border-red-200 bg-red-50 text-red-800 rounded-lg p-4 text-sm">
            Failed to load load test result: {String(error)}
          </div>
        )}

        {result && (
          <LoadTestReportContent
            result={result}
            title={timestamp ? formatLoadTestTimestamp(timestamp) : "Load test"}
            subtitle={
              <>
                Network: <span className="font-mono">{network}</span>
                {timestamp && (
                  <>
                    {" · "}
                    <span className="font-mono text-slate-400">
                      {timestamp}
                    </span>
                  </>
                )}
              </>
            }
            backLink={{
              to: `/load-tests/${network ?? "sepolia"}/all`,
              label: "View all runs →",
            }}
            network={network}
          />
        )}
      </main>
    </div>
  );
};

export default LoadTestDetail;
