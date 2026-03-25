import type { MetricsResponse } from "@/lib/api";

type Props = {
  metrics: MetricsResponse | null;
  loading: boolean;
  lastUpdated: Date | null;
};

function Card({
  label,
  value,
  sub,
}: {
  label: string;
  value: string;
  sub?: string;
}) {
  return (
    <div className="rounded-lg border border-surface-border bg-surface-raised/60 px-4 py-3">
      <p className="text-xs font-medium uppercase tracking-wide text-zinc-500">
        {label}
      </p>
      <p className="mt-1 font-mono text-2xl font-semibold tabular-nums text-white">
        {value}
      </p>
      {sub && <p className="mt-0.5 text-xs text-zinc-500">{sub}</p>}
    </div>
  );
}

export function MetricStrip({ metrics, loading, lastUpdated }: Props) {
  const fmt = (n: number | undefined) =>
    loading && n == null ? "…" : (n ?? 0).toLocaleString();

  return (
    <div className="space-y-2">
      <div className="grid gap-3 sm:grid-cols-3">
        <Card
          label="Transactions"
          value={fmt(metrics?.total_transactions)}
          sub="All time in database"
        />
        <Card
          label="Open alerts"
          value={fmt(metrics?.open_alerts)}
          sub="Requires review"
        />
        <Card
          label="Total alerts"
          value={fmt(metrics?.total_alerts)}
          sub="Including resolved"
        />
      </div>
      {lastUpdated && (
        <p className="text-right text-xs text-zinc-600">
          Updated {lastUpdated.toLocaleTimeString()}
        </p>
      )}
    </div>
  );
}
