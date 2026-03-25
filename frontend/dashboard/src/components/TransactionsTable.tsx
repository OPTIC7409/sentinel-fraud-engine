import type { TransactionRow } from "@/lib/api";
import { formatMoney, formatTime, riskTone } from "@/lib/format";

type Props = {
  rows: TransactionRow[];
  loading: boolean;
};

export function TransactionsTable({ rows, loading }: Props) {
  return (
    <div className="overflow-hidden rounded-xl border border-surface-border bg-surface-raised/40">
      <div className="border-b border-surface-border px-4 py-3">
        <h2 className="text-sm font-semibold text-white">Recent transactions</h2>
        <p className="text-xs text-zinc-500">
          Newest first, joined with risk scores when available
        </p>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[640px] text-left text-sm">
          <thead>
            <tr className="border-b border-surface-border text-xs uppercase tracking-wide text-zinc-500">
              <th className="px-4 py-2 font-medium">Time</th>
              <th className="px-4 py-2 font-medium">User</th>
              <th className="px-4 py-2 font-medium">Amount</th>
              <th className="px-4 py-2 font-medium">Merchant</th>
              <th className="px-4 py-2 font-medium">Category</th>
              <th className="px-4 py-2 font-medium text-right">Risk</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-surface-border">
            {loading && rows.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-zinc-500">
                  Loading…
                </td>
              </tr>
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-zinc-500">
                  No transactions yet. Run the load test or ingest pipeline.
                </td>
              </tr>
            ) : (
              rows.map((t) => {
                const r = riskTone(t.risk_score);
                return (
                  <tr
                    key={t.id}
                    className="transition hover:bg-white/[0.02]"
                  >
                    <td className="whitespace-nowrap px-4 py-2.5 font-mono text-xs text-zinc-400">
                      {formatTime(t.timestamp)}
                    </td>
                    <td className="px-4 py-2.5 font-mono text-xs text-zinc-300">
                      {t.user_id}
                    </td>
                    <td className="whitespace-nowrap px-4 py-2.5 font-medium text-zinc-100">
                      {formatMoney(t.amount, t.currency)}
                    </td>
                    <td className="max-w-[140px] truncate px-4 py-2.5 font-mono text-xs text-zinc-400">
                      {t.merchant_id}
                    </td>
                    <td className="px-4 py-2.5 text-zinc-400">
                      {t.merchant_category}
                    </td>
                    <td className={`px-4 py-2.5 text-right font-mono text-sm ${r.className}`}>
                      {r.label}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
