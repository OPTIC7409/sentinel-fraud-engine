"use client";

import type { AlertRow } from "@/lib/api";
import { formatTime } from "@/lib/format";

type Props = {
  alerts: AlertRow[];
  loading: boolean;
  onResolve: (id: number) => Promise<void>;
  resolvingId: number | null;
};

export function AlertsPanel({
  alerts,
  loading,
  onResolve,
  resolvingId,
}: Props) {
  return (
    <div className="overflow-hidden rounded-xl border border-surface-border bg-surface-raised/40">
      <div className="border-b border-surface-border px-4 py-3">
        <h2 className="text-sm font-semibold text-white">Open alerts</h2>
        <p className="text-xs text-zinc-500">High-risk items from the alert service</p>
      </div>
      <ul className="divide-y divide-surface-border">
        {loading && alerts.length === 0 ? (
          <li className="px-4 py-8 text-center text-sm text-zinc-500">
            Loading…
          </li>
        ) : alerts.length === 0 ? (
          <li className="px-4 py-8 text-center text-sm text-zinc-500">
            No open alerts. System is quiet.
          </li>
        ) : (
          alerts.map((a) => (
            <li
              key={a.id}
              className="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
            >
              <div>
                <div className="flex flex-wrap items-baseline gap-2">
                  <span className="font-mono text-xs text-zinc-500">
                    #{a.id}
                  </span>
                  <span
                    className={
                      a.risk_score >= 75
                        ? "text-risk-critical font-semibold"
                        : "text-risk-high"
                    }
                  >
                    Risk {a.risk_score}
                  </span>
                  <span className="text-xs uppercase text-zinc-500">
                    {a.priority} priority
                  </span>
                </div>
                <p className="mt-1 font-mono text-xs text-zinc-400">
                  Txn {a.transaction_id}
                </p>
                <p className="text-xs text-zinc-600">
                  {formatTime(a.created_at)}
                </p>
              </div>
              <button
                type="button"
                onClick={() => onResolve(a.id)}
                disabled={resolvingId === a.id}
                className="shrink-0 rounded-lg border border-surface-border px-3 py-1.5 text-xs font-medium text-zinc-200 transition hover:border-zinc-500 hover:bg-white/5 disabled:opacity-40"
              >
                {resolvingId === a.id ? "Resolving…" : "Resolve"}
              </button>
            </li>
          ))
        )}
      </ul>
    </div>
  );
}
