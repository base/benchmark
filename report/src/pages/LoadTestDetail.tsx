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
  formatEthFromWei,
  formatGasVerbose,
  formatGpsVerbose,
  formatLoadTestTimestamp,
  formatPercent,
  formatTps,
} from "../utils/formatters";
import {
  BlockRange,
  FlashblocksLatencyStats,
  LatencyStats,
  LoadTestResult,
  ObservedWindowMetrics,
  TailMetrics,
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

const ObservedWindowSummary = ({
  window,
}: {
  window: ObservedWindowMetrics;
}) => {
  const blockRange = window.block_range;

  return (
    <StatCard title="Observed window">
      <StatGrid>
        <Stat
          label="Window duration"
          value={formatDuration(window.duration)}
          hint={`${window.expected_block_count.toLocaleString()} expected blocks`}
        />
        <Stat
          label="Confirmed in window"
          value={window.confirmed_count.toLocaleString()}
        />
        <Stat label="TPS" value={formatTps(window.tps)} />
        <Stat label="Gas/s" value={formatGpsVerbose(window.gps)} />
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

const TailSection = ({
  tail,
  totalConfirmed,
}: {
  tail: TailMetrics;
  totalConfirmed: number;
}) => {
  const blockRange = tail.block_range;
  const hasReceiptDelay =
    tail.block_receipt_delay &&
    durationToNanos(tail.block_receipt_delay.max) > 0;

  const timePastRows = useMemo(
    () => buildLatencyRows(tail.time_past_observed_window),
    [tail.time_past_observed_window],
  );
  const blockLatencyRows = useMemo(
    () => buildLatencyRows(tail.block_latency),
    [tail.block_latency],
  );
  const receiptDelayRows = useMemo(
    () => (hasReceiptDelay ? buildLatencyRows(tail.block_receipt_delay) : []),
    [tail.block_receipt_delay, hasReceiptDelay],
  );
  const flashblocksRows = useMemo(
    () => buildLatencyRows(tail.flashblocks_latency),
    [tail.flashblocks_latency],
  );

  if (tail.count === 0) {
    return (
      <StatCard title="Tail inclusion (txs past the observed window)">
        <div className="text-sm text-slate-500">
          No transactions landed past the observed window
          {typeof tail.observed_window_end_block === "number" && (
            <>
              {" "}
              (boundary: block {tail.observed_window_end_block.toLocaleString()}
              )
            </>
          )}
          .
        </div>
      </StatCard>
    );
  }

  return (
    <StatCard title="Tail inclusion (txs past the observed window)">
      <div className="flex flex-col gap-y-6">
        <StatGrid>
          <Stat
            label="Tail count"
            value={tail.count.toLocaleString()}
            hint={`of ${totalConfirmed.toLocaleString()} confirmed`}
          />
          <Stat
            label="% of confirmed"
            value={`${tail.confirmed_pct.toFixed(2)}%`}
          />
          {typeof tail.observed_window_end_block === "number" && (
            <Stat
              label="Window end block"
              value={tail.observed_window_end_block.toLocaleString()}
              hint="tail = block_number > end"
            />
          )}
          {blockRange && (
            <Stat
              label="Tail block range"
              value={formatBlockRange(blockRange)}
              hint={`${blockRange.block_count.toLocaleString()} blocks`}
            />
          )}
        </StatGrid>

        <div>
          <h3 className="text-xs uppercase tracking-wide text-slate-500 mb-2">
            Time past observed window
          </h3>
          <PercentileBarChart rows={timePastRows} barColorClass="bg-rose-500" />
        </div>

        <div>
          <h3 className="text-xs uppercase tracking-wide text-slate-500 mb-2">
            Block latency (tail)
          </h3>
          <PercentileBarChart
            rows={blockLatencyRows}
            barColorClass="bg-amber-500"
          />
        </div>

        {hasReceiptDelay && (
          <div>
            <h3 className="text-xs uppercase tracking-wide text-slate-500 mb-2">
              Block receipt delay (tail)
            </h3>
            <PercentileBarChart
              rows={receiptDelayRows}
              barColorClass="bg-sky-500"
            />
          </div>
        )}

        <div>
          <h3 className="text-xs uppercase tracking-wide text-slate-500 mb-2">
            Flashblocks latency (tail) ·{" "}
            {tail.flashblocks_latency.count.toLocaleString()} samples
          </h3>
          <PercentileBarChart
            rows={flashblocksRows}
            barColorClass="bg-fuchsia-500"
          />
        </div>
      </div>
    </StatCard>
  );
};

const FullRunBaselineSection = ({ result }: { result: LoadTestResult }) => {
  const submitted = result.throughput.total_submitted;
  const confirmed = result.throughput.total_confirmed;
  const failed = result.throughput.total_failed;
  const reverted = result.throughput.total_reverted;
  const blockRange = result.block_range;

  const blockLatencyRows = useMemo(
    () => buildLatencyRows(result.block_latency),
    [result.block_latency],
  );
  const flashblocksRows = useMemo(
    () => buildLatencyRows(result.flashblocks_latency),
    [result.flashblocks_latency],
  );
  const receiptDelayRows = useMemo(
    () =>
      result.block_receipt_delay
        ? buildLatencyRows(result.block_receipt_delay)
        : [],
    [result.block_receipt_delay],
  );

  return (
    <details className="border border-slate-200 bg-white rounded-lg">
      <summary className="cursor-pointer select-none px-6 py-4 text-sm font-semibold text-slate-500 uppercase tracking-wide hover:bg-slate-50">
        Full-run baseline (observed window + tail combined)
      </summary>
      <div className="px-6 pb-6 pt-2 flex flex-col gap-y-6">
        <p className="text-xs text-slate-500 -mt-2">
          Full-run averages dilute the clean reporting window with tail
          stragglers. Use the observed-window numbers above for headline
          comparisons; this section is included for completeness.
        </p>

        <StatGrid>
          <Stat
            label="Wall-clock duration"
            value={formatDuration(result.throughput.duration)}
          />
          <Stat label="Submitted" value={submitted.toLocaleString()} />
          <Stat
            label="Confirmed"
            value={confirmed.toLocaleString()}
            hint={formatPercent(confirmed, submitted) + " of submitted"}
          />
          <Stat label="Failed" value={failed.toLocaleString()} />
          {reverted > 0 && (
            <Stat
              label="Reverted"
              value={reverted.toLocaleString()}
              hint={formatPercent(reverted, confirmed) + " of confirmed"}
            />
          )}
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
              value={formatBlockRange(blockRange)}
              hint={`${blockRange.block_count.toLocaleString()} blocks`}
            />
          )}
        </StatGrid>

        <div>
          <h3 className="text-xs uppercase tracking-wide text-slate-500 mb-2">
            Block latency (full run)
          </h3>
          <PercentileBarChart
            rows={blockLatencyRows}
            barColorClass="bg-amber-500"
          />
        </div>

        {result.block_receipt_delay && (
          <div>
            <h3 className="text-xs uppercase tracking-wide text-slate-500 mb-2">
              Block receipt delay (full run)
            </h3>
            <PercentileBarChart
              rows={receiptDelayRows}
              barColorClass="bg-sky-500"
            />
          </div>
        )}

        <div>
          <h3 className="text-xs uppercase tracking-wide text-slate-500 mb-2">
            Flashblocks latency (full run) ·{" "}
            {result.flashblocks_latency.count.toLocaleString()} samples
          </h3>
          <PercentileBarChart
            rows={flashblocksRows}
            barColorClass="bg-fuchsia-500"
          />
        </div>
      </div>
    </details>
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
}

export const LoadTestReportContent = ({
  result,
  title,
  subtitle,
  backLink,
}: LoadTestReportContentProps) => {
  const observedWindow = result.observed_window;
  const tail = result.tail ?? undefined;

  // Headline numbers come from observed_window when available, otherwise fall
  // back to the legacy full-run fields so older S3 runs still render.
  const headlineTps = observedWindow?.tps ?? result.throughput.tps;
  const headlineBlockLatency =
    observedWindow?.block_latency ?? result.block_latency;
  const headlineFlashblocksLatency =
    observedWindow?.flashblocks_latency ?? result.flashblocks_latency;
  const headlineReceiptDelay =
    observedWindow?.block_receipt_delay ?? result.block_receipt_delay;

  const headlineBlockLatencyRows = useMemo(
    () => buildLatencyRows(headlineBlockLatency),
    [headlineBlockLatency],
  );
  const headlineFlashblocksRows = useMemo(
    () => buildLatencyRows(headlineFlashblocksLatency),
    [headlineFlashblocksLatency],
  );
  const headlineReceiptDelayRows = useMemo(
    () => (headlineReceiptDelay ? buildLatencyRows(headlineReceiptDelay) : []),
    [headlineReceiptDelay],
  );

  const headlineLabel = observedWindow ? "Observed-window TPS" : "Swaps/s";
  const latencyScopeLabel = observedWindow ? "observed window" : "full run";

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

      {observedWindow && <ObservedWindowSummary window={observedWindow} />}

      <StatCard title={`Block latency (submit → block, ${latencyScopeLabel})`}>
        <PercentileBarChart
          rows={headlineBlockLatencyRows}
          barColorClass="bg-amber-500"
        />
      </StatCard>

      {headlineReceiptDelay && (
        <StatCard
          title={`Block receipt delay (block → receipt, ${latencyScopeLabel})`}
        >
          <PercentileBarChart
            rows={headlineReceiptDelayRows}
            barColorClass="bg-sky-500"
          />
        </StatCard>
      )}

      <StatCard
        title={`Flashblocks latency (submit → flashblock, ${latencyScopeLabel}) · ${headlineFlashblocksLatency.count.toLocaleString()} samples`}
      >
        <PercentileBarChart
          rows={headlineFlashblocksRows}
          barColorClass="bg-fuchsia-500"
        />
      </StatCard>

      {tail && (
        <TailSection
          tail={tail}
          totalConfirmed={result.throughput.total_confirmed}
        />
      )}

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

      {observedWindow && <FullRunBaselineSection result={result} />}
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
          />
        )}
      </main>
    </div>
  );
};

export default LoadTestDetail;
