import { LoadTestConfig } from "../types";
import { formatEthFromWeiString } from "../utils/formatters";
import StatCard from "./StatCard";

interface ConfigCardProps {
  config: LoadTestConfig;
}

interface Row {
  label: string;
  value: string;
}

const formatTransactions = (txs: LoadTestConfig["transactions"]): string => {
  if (!txs || txs.length === 0) return "—";
  const total = txs.reduce((acc, t) => acc + t.weight, 0);
  if (total === 0) return txs.map((t) => t.type).join(" · ");
  return txs
    .map((t) => `${t.type} (${Math.round((t.weight / total) * 100)}%)`)
    .join(" · ");
};

const formatTargetGps = (gps: number): string => {
  if (gps >= 1e9) return `${(gps / 1e9).toFixed(1)}B gas/s`;
  if (gps >= 1e6) return `${(gps / 1e6).toFixed(0)}M gas/s`;
  if (gps >= 1e3) return `${(gps / 1e3).toFixed(0)}k gas/s`;
  return `${gps.toLocaleString()} gas/s`;
};

const buildRows = (config: LoadTestConfig): Row[][] => {
  const loadShape: Row[] = [
    { label: "Senders", value: config.sender_count.toLocaleString() },
    {
      label: "In-flight / sender",
      value: config.in_flight_per_sender.toLocaleString(),
    },
    { label: "Batch size", value: config.batch_size.toLocaleString() },
    { label: "Batch timeout", value: config.batch_timeout },
  ];
  if (config.sender_offset !== 0) {
    loadShape.push({
      label: "Sender offset",
      value: config.sender_offset.toLocaleString(),
    });
  }

  const target: Row[] = [
    { label: "Duration", value: config.duration },
    { label: "Target gas/s", value: formatTargetGps(config.target_gps) },
  ];

  const funding: Row[] = [
    {
      label: "Funding / sender",
      value: formatEthFromWeiString(config.funding_amount),
    },
  ];
  const hasSwapToken =
    config.transactions.some((t) => t.type === "swap") &&
    config.swap_token_amount &&
    config.swap_token_amount !== "0";
  if (hasSwapToken) {
    funding.push({
      label: "Swap token amount",
      value: formatEthFromWeiString(config.swap_token_amount),
    });
  }

  const repro: Row[] = [{ label: "Seed", value: config.seed.toLocaleString() }];
  if (config.chain_id !== null) {
    repro.push({ label: "Chain ID", value: config.chain_id.toLocaleString() });
  }
  if (config.looper_contract) {
    repro.push({ label: "Looper contract", value: config.looper_contract });
  }

  return [loadShape, target, funding, repro];
};

const RowGroup = ({ rows }: { rows: Row[] }) => (
  <div className="grid grid-cols-2 gap-x-6 gap-y-3">
    {rows.map((r) => (
      <div key={r.label} className="flex flex-col">
        <span className="text-xs uppercase tracking-wide text-slate-500">
          {r.label}
        </span>
        <span className="text-sm text-slate-900 font-mono mt-0.5 break-all">
          {r.value}
        </span>
      </div>
    ))}
  </div>
);

const ConfigCard = ({ config }: ConfigCardProps) => {
  const groups = buildRows(config);
  const txLine = formatTransactions(config.transactions);

  return (
    <StatCard title="Run config">
      <div className="flex flex-col gap-y-5">
        {groups.map((rows, i) => (
          <div key={i}>
            {i > 0 && <hr className="border-slate-100 mb-5" />}
            <RowGroup rows={rows} />
          </div>
        ))}
        <hr className="border-slate-100" />
        <div className="flex flex-col">
          <span className="text-xs uppercase tracking-wide text-slate-500">
            Workload
          </span>
          <span className="text-sm text-slate-900 mt-0.5">{txLine}</span>
        </div>
      </div>
    </StatCard>
  );
};

export default ConfigCard;
