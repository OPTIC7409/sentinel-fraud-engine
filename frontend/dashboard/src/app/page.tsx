import Link from "next/link";

export default function HomePage() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center px-6">
      <div className="max-w-lg text-center">
        <p className="font-mono text-xs uppercase tracking-[0.2em] text-zinc-500">
          Sentinel Fraud Engine
        </p>
        <h1 className="mt-4 text-3xl font-semibold tracking-tight text-white sm:text-4xl">
          Operations dashboard
        </h1>
        <p className="mt-3 text-sm leading-relaxed text-zinc-400">
          Monitor scored transactions, open alerts, and aggregate metrics against
          the API gateway.
        </p>
        <div className="mt-10 flex justify-center">
          <Link
            href="/login"
            className="rounded-lg bg-accent px-5 py-2.5 text-sm font-medium text-white shadow-lg shadow-blue-500/20 transition hover:bg-accent-muted"
          >
            Sign in
          </Link>
        </div>
        <p className="mt-12 text-xs text-zinc-600">
          Ensure the stack is running:{" "}
          <code className="rounded bg-surface-raised px-1.5 py-0.5 font-mono text-zinc-400">
            ./start.sh
          </code>
        </p>
      </div>
    </main>
  );
}
