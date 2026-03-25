"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  clearStoredToken,
  fetchAlerts,
  fetchMetrics,
  fetchTransactions,
  getStoredToken,
  resolveAlert,
  type AlertRow,
  type MetricsResponse,
  type TransactionRow,
} from "@/lib/api";
import { AlertsPanel } from "@/components/AlertsPanel";
import { MetricStrip } from "@/components/MetricStrip";
import { TransactionsTable } from "@/components/TransactionsTable";

const POLL_MS = 5000;

export default function DashboardPage() {
  const router = useRouter();
  const [transactions, setTransactions] = useState<TransactionRow[]>([]);
  const [alerts, setAlerts] = useState<AlertRow[]>([]);
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [resolvingId, setResolvingId] = useState<number | null>(null);

  const load = useCallback(async () => {
    if (!getStoredToken()) {
      router.replace("/login");
      return;
    }
    try {
      const [tx, al, m] = await Promise.all([
        fetchTransactions(40),
        fetchAlerts("open", 30),
        fetchMetrics(),
      ]);
      setTransactions(tx.transactions);
      setAlerts(al.alerts);
      setMetrics(m);
      setError(null);
      setLastUpdated(new Date());
    } catch (e) {
      if (e instanceof Error && e.message === "unauthorized") {
        clearStoredToken();
        router.replace("/login");
        return;
      }
      setError(e instanceof Error ? e.message : "Failed to refresh");
    } finally {
      setLoading(false);
    }
  }, [router]);

  useEffect(() => {
    load();
    const id = window.setInterval(load, POLL_MS);
    return () => window.clearInterval(id);
  }, [load]);

  async function handleResolve(id: number) {
    setResolvingId(id);
    try {
      await resolveAlert(id);
      await load();
    } catch (e) {
      if (e instanceof Error && e.message === "unauthorized") {
        clearStoredToken();
        router.replace("/login");
        return;
      }
      setError(e instanceof Error ? e.message : "Resolve failed");
    } finally {
      setResolvingId(null);
    }
  }

  function logout() {
    clearStoredToken();
    router.replace("/login");
  }

  return (
    <div className="min-h-screen px-4 pb-16 pt-8 sm:px-8">
      <header className="mx-auto flex max-w-6xl flex-col gap-4 border-b border-surface-border pb-6 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="font-mono text-xs uppercase tracking-[0.2em] text-zinc-500">
            Sentinel
          </p>
          <h1 className="mt-1 text-2xl font-semibold tracking-tight text-white">
            Operations
          </h1>
          <p className="mt-1 max-w-xl text-sm text-zinc-400">
            Live view of PostgreSQL-backed transactions and alerts. Refreshes
            every {POLL_MS / 1000}s.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => load()}
            className="rounded-lg border border-surface-border px-3 py-2 text-sm text-zinc-200 transition hover:bg-white/5"
          >
            Refresh now
          </button>
          <button
            type="button"
            onClick={logout}
            className="rounded-lg border border-surface-border px-3 py-2 text-sm text-zinc-400 transition hover:text-white"
          >
            Sign out
          </button>
          <Link
            href="/"
            className="rounded-lg px-3 py-2 text-sm text-zinc-500 hover:text-zinc-300"
          >
            Home
          </Link>
        </div>
      </header>

      <div className="mx-auto mt-8 max-w-6xl space-y-8">
        {error && (
          <div className="rounded-lg border border-amber-500/30 bg-amber-950/30 px-4 py-3 text-sm text-amber-100">
            {error}
          </div>
        )}

        <MetricStrip
          metrics={metrics}
          loading={loading}
          lastUpdated={lastUpdated}
        />

        <div className="grid gap-8 lg:grid-cols-5">
          <div className="lg:col-span-3">
            <TransactionsTable rows={transactions} loading={loading} />
          </div>
          <div className="lg:col-span-2">
            <AlertsPanel
              alerts={alerts}
              loading={loading}
              onResolve={handleResolve}
              resolvingId={resolvingId}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
