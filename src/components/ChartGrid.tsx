import React, { useMemo } from "react";
import { SORTED_CHART_CONFIG } from "../metricDefinitions";
import { DataSeries } from "../types";
import { formatValue } from "../utils/formatters";
import LineChart from "./LineChart";

interface ProvidedProps {
  data: DataSeries[];
  role: "sequencer" | "validator" | null;
}

interface ChartSection {
  title: string;
  description: string;
  metrics: string[];
}

const SERIES_COLORS = [
  "#1f77b4",
  "#ff7f0e",
  "#2ca02c",
  "#d62728",
  "#9467bd",
  "#8c564b",
  "#e377c2",
  "#7f7f7f",
  "#bcbd22",
  "#17becf",
];

const OVERVIEW_METRICS = {
  sequencer: [
    "gas/per_second",
    "transactions/per_second",
    "reth_base_builder_total_block_built_duration_avg",
    "reth_base_builder_state_root_calculation_duration_avg",
  ],
  validator: [
    "gas/per_second",
    "transactions/per_second",
    "reth_consensus_engine_beacon_new_payload_latency_avg",
    "reth_tree_root_sparse_trie_total_duration_histogram_avg",
  ],
} as const;

const ROLE_FOCUS_METRIC = {
  sequencer: {
    key: "reth_base_builder_total_block_built_duration_avg",
    label: "Avg. block build",
    description: "Time spent assembling a block",
  },
  validator: {
    key: "reth_consensus_engine_beacon_new_payload_latency_avg",
    label: "Avg. new payload",
    description: "Time spent processing each payload",
  },
} as const;

const COMMON_SECTIONS: ChartSection[] = [
  {
    title: "Block profile",
    description:
      "Transaction density gives context to throughput without exposing internal scrape data.",
    metrics: ["transactions/per_block"],
  },
];

const ROLE_SECTIONS: Record<"sequencer" | "validator", ChartSection[]> = {
  sequencer: [
    {
      title: "Builder pipeline",
      description: "Where the sequencer spends time while assembling a block.",
      metrics: [
        "reth_base_builder_flashblock_build_duration_avg",
        "reth_base_builder_sequencer_tx_duration_avg",
        "reth_base_builder_payload_transaction_simulation_duration_avg",
        "reth_base_builder_tx_simulation_duration_avg",
        "reth_base_builder_payload_num_tx_gauge",
        "reth_base_builder_flashblock_count",
      ],
    },
    {
      title: "Execution and validation",
      description:
        "Canonical execution and state-validation work that can affect sequencer throughput.",
      metrics: [
        "reth_sync_execution_execution_duration_avg",
        "reth_sync_block_validation_total_duration_avg",
        "reth_sync_block_validation_state_root_duration_avg",
        "reth_sync_block_validation_deferred_trie_compute_duration_avg",
      ],
    },
    {
      title: "Trie, persistence, and queue",
      description:
        "Storage-path metrics to inspect after a builder or state-processing regression.",
      metrics: [
        "reth_tree_root_sparse_trie_total_duration_histogram_avg",
        "reth_tree_root_sparse_trie_final_update_duration_histogram_avg",
        "reth_tree_root_sparse_trie_cache_wait_duration_histogram_avg",
        "reth_consensus_engine_persistence_save_blocks_duration_seconds_avg",
        "reth_consensus_engine_persistence_save_blocks_batch_size_avg",
        "reth_transaction_pool_pending_pool_transactions",
        "reth_transaction_pool_total_transactions",
        "reth_db_freelist",
      ],
    },
  ],
  validator: [
    {
      title: "Validator pipeline",
      description: "Execution, validation, and durable block-insertion work.",
      metrics: [
        "reth_sync_execution_execution_duration_avg",
        "reth_sync_block_validation_total_duration_avg",
        "reth_sync_block_validation_state_root_duration_avg",
        "reth_sync_block_validation_deferred_trie_compute_duration_avg",
        "reth_consensus_engine_beacon_block_insert_total_duration_avg",
        "reth_consensus_engine_persistence_save_blocks_duration_seconds_avg",
      ],
    },
    {
      title: "Trie and state",
      description:
        "State-processing detail for explaining validation bottlenecks.",
      metrics: [
        "reth_tree_root_sparse_trie_final_update_duration_histogram_avg",
        "reth_tree_root_sparse_trie_cache_wait_duration_histogram_avg",
        "reth_tree_root_sparse_trie_channel_wait_duration_histogram_avg",
        "reth_tree_root_sparse_trie_reveal_multiproof_duration_histogram_avg",
        "reth_sync_state_provider_total_storage_fetch_latency_avg",
        "reth_sync_state_provider_total_code_fetch_latency_avg",
        "reth_sync_state_provider_total_account_fetch_latency_avg",
        "reth_db_freelist",
      ],
    },
    {
      title: "Execution workload and engine health",
      description:
        "Per-transaction work, state growth, and engine pressure behind validation behavior.",
      metrics: [
        "reth_sync_execution_transaction_execution_histogram_avg",
        "reth_sync_execution_transaction_wait_histogram_avg",
        "reth_sync_execution_accounts_updated_histogram_avg",
        "reth_sync_execution_storage_slots_updated_histogram_avg",
        "reth_sync_block_validation_hashed_post_state_size_avg",
        "reth_sync_block_validation_trie_updates_sorted_size_avg",
        "reth_consensus_engine_persistence_save_blocks_batch_size_avg",
        "reth_consensus_engine_beacon_backpressure_active",
        "reth_transaction_pool_pending_pool_transactions",
        "reth_transaction_pool_total_transactions",
      ],
    },
  ],
};

function resolveMetricKey(
  data: DataSeries[],
  primaryKey: string,
  aliases: string[] = [],
): string {
  const keys = [primaryKey, ...aliases];
  const chartData = data.flatMap((series) => series.data);
  for (const key of keys) {
    if (
      chartData.some((sample) => sample.ExecutionMetrics[key] !== undefined)
    ) {
      return key;
    }
  }

  const metricKeys = chartData.flatMap((sample) =>
    Object.keys(sample.ExecutionMetrics),
  );
  for (const key of keys) {
    const quantileSuffix = key.match(/(_quantile_\d+(?:_\d+)?)$/)?.[1] ?? "";
    const metricPrefix = quantileSuffix
      ? key.slice(0, -quantileSuffix.length)
      : key;
    const labeledMetricKeys = metricKeys.filter(
      (metricKey) =>
        metricKey.startsWith(`${metricPrefix}_`) &&
        (!quantileSuffix || metricKey.endsWith(quantileSuffix)),
    );
    if (labeledMetricKeys.length === 1) return labeledMetricKeys[0];
  }
  return primaryKey;
}

const averageMetric = (
  data: DataSeries[],
  metricKey: string,
): number | undefined => {
  // Average each run first, so a long benchmark does not outweigh shorter
  // selected runs in the comparison headline.
  const perRunAverages = data.flatMap((series) => {
    const values = series.data
      .map((sample) => sample.ExecutionMetrics[metricKey])
      .filter(
        (value): value is number =>
          typeof value === "number" && Number.isFinite(value),
      );
    return values.length > 0
      ? [values.reduce((total, value) => total + value, 0) / values.length]
      : [];
  });
  if (perRunAverages.length === 0) return undefined;
  return (
    perRunAverages.reduce((total, value) => total + value, 0) /
    perRunAverages.length
  );
};

const ChartGrid: React.FC<ProvidedProps> = ({ data, role }) => {
  const availableCharts = useMemo(() => {
    const chartData = data.flatMap((series) => series.data);
    return new Map(
      SORTED_CHART_CONFIG.flatMap(([metricKey, config]) => {
        const resolvedKey = resolveMetricKey(data, metricKey, config.aliases);
        return chartData.some(
          (sample) => sample.ExecutionMetrics[resolvedKey] !== undefined,
        )
          ? [[metricKey, { config, resolvedKey }] as const]
          : [];
      }),
    );
  }, [data]);

  if (!role || data.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
        Select a node role and one or more benchmark runs to compare metrics.
      </div>
    );
  }

  const renderChart = (metricKey: string, emphasis = false) => {
    const chart = availableCharts.get(metricKey);
    if (!chart) return null;

    return (
      <article
        key={metricKey}
        className={`chart-container border-slate-200 bg-white p-3 shadow-sm ${
          emphasis ? "ring-1 ring-blue-100" : ""
        }`}
      >
        <LineChart
          thresholdKey={`${role}/${metricKey}`}
          series={data}
          metricKey={chart.resolvedKey}
          title={chart.config.title}
          description={chart.config.description}
          unit={chart.config.unit}
          thresholds={data[0]?.thresholds}
        />
      </article>
    );
  };

  const overviewMetrics = OVERVIEW_METRICS[role];
  const sections = [...COMMON_SECTIONS, ...ROLE_SECTIONS[role]];
  const focusMetric = ROLE_FOCUS_METRIC[role];
  const averageGasPerSecond = averageMetric(data, "gas/per_second");
  const averageTransactionsPerSecond = averageMetric(
    data,
    "transactions/per_second",
  );
  const averageRoleProcessingTime = averageMetric(data, focusMetric.key);
  const overviewStats = [
    {
      label: "Average TPS",
      value:
        averageTransactionsPerSecond === undefined
          ? "—"
          : `${averageTransactionsPerSecond.toLocaleString(undefined, {
              maximumFractionDigits: 1,
            })} TPS`,
      description:
        "Equal-weight mean sustained throughput across selected runs",
      className:
        "border-blue-200 bg-gradient-to-br from-blue-600 to-blue-700 text-white shadow-blue-200/70 sm:col-span-2 xl:col-span-5",
      labelClassName: "text-blue-100",
      valueClassName: "text-4xl text-white sm:text-5xl",
      descriptionClassName: "text-blue-100",
    },
    {
      label: "Average Gas/s",
      value:
        averageGasPerSecond === undefined
          ? "—"
          : formatValue(averageGasPerSecond, "gas/s"),
      description:
        "Equal-weight mean canonical throughput across selected runs",
      className: "border-slate-200 bg-white xl:col-span-3",
      labelClassName: "text-slate-500",
      valueClassName: "text-2xl text-slate-900",
      descriptionClassName: "text-slate-500",
    },
    {
      label: focusMetric.label,
      value:
        averageRoleProcessingTime === undefined
          ? "—"
          : formatValue(averageRoleProcessingTime, "s"),
      description: focusMetric.description,
      className: "border-slate-200 bg-white xl:col-span-2",
      labelClassName: "text-slate-500",
      valueClassName: "text-2xl text-slate-900",
      descriptionClassName: "text-slate-500",
    },
    {
      label: "Runs compared",
      value: data.length.toLocaleString(),
      description: "One colored line per run",
      className: "border-slate-200 bg-white xl:col-span-2",
      labelClassName: "text-slate-500",
      valueClassName: "text-2xl text-slate-900",
      descriptionClassName: "text-slate-500",
    },
  ];

  return (
    <div className="space-y-12 pb-12">
      <section aria-labelledby="overview-heading">
        <div className="mb-5 flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
              Snapshot benchmark
            </p>
            <h2
              id="overview-heading"
              className="mt-1 text-2xl font-semibold text-slate-900"
            >
              Performance overview
            </h2>
            <p className="mt-1 text-sm text-slate-500">
              Throughput and the role-specific timings most likely to explain
              it.
            </p>
          </div>
          <span className="w-fit rounded-full bg-slate-100 px-3 py-1 text-xs font-medium capitalize text-slate-600">
            {role} view
          </span>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-12">
          {overviewStats.map((stat) => (
            <div
              key={stat.label}
              className={`rounded-xl border px-5 py-4 shadow-sm ${stat.className}`}
            >
              <p
                className={`text-xs font-semibold uppercase tracking-[0.12em] ${stat.labelClassName}`}
              >
                {stat.label}
              </p>
              <p
                className={`mt-1 font-semibold tracking-tight tabular-nums ${stat.valueClassName}`}
              >
                {stat.value}
              </p>
              <p className={`mt-1 text-xs ${stat.descriptionClassName}`}>
                {stat.description}
              </p>
            </div>
          ))}
        </div>
        <div className="mt-4 flex flex-col gap-2 rounded-xl border border-slate-200 bg-white px-4 py-3 sm:flex-row sm:items-center">
          <span className="shrink-0 text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
            Compared runs
          </span>
          <div className="flex flex-wrap gap-x-4 gap-y-2">
            {data.map((series, index) => (
              <span
                key={`${series.name}-${index}`}
                className="inline-flex items-center gap-2 text-xs text-slate-700"
              >
                <span
                  className="h-2.5 w-2.5 rounded-full"
                  style={{
                    backgroundColor:
                      series.color ??
                      SERIES_COLORS[index % SERIES_COLORS.length],
                  }}
                />
                <span>{series.name}</span>
              </span>
            ))}
          </div>
        </div>
        <div className="mt-5 grid grid-cols-1 gap-5 xl:grid-cols-2">
          {overviewMetrics.map((metricKey) => renderChart(metricKey, true))}
        </div>
      </section>

      {sections.map((section) => {
        const charts = section.metrics.map((metricKey) =>
          renderChart(metricKey),
        );
        if (charts.every((chart) => chart === null)) return null;

        return (
          <section
            key={section.title}
            aria-labelledby={`${section.title}-heading`}
          >
            <div className="mb-4 border-b border-slate-200 pb-3">
              <h2
                id={`${section.title}-heading`}
                className="text-lg font-semibold text-slate-900"
              >
                {section.title}
              </h2>
              <p className="mt-1 text-sm text-slate-500">
                {section.description}
              </p>
            </div>
            <div className="grid grid-cols-1 gap-5 xl:grid-cols-2">
              {charts}
            </div>
          </section>
        );
      })}
    </div>
  );
};

export default ChartGrid;
