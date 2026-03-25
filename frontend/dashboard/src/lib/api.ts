const TOKEN_KEY = "sentinel_jwt";

export function getApiBase(): string {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8000";
  return base.replace(/\/$/, "");
}

export function getStoredToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(TOKEN_KEY);
}

export function setStoredToken(token: string): void {
  window.localStorage.setItem(TOKEN_KEY, token);
}

export function clearStoredToken(): void {
  window.localStorage.removeItem(TOKEN_KEY);
}

export type LoginResponse = {
  token: string;
  user: { email: string; role: string };
};

export type TransactionRow = {
  id: string;
  user_id: string;
  amount: number;
  currency: string;
  merchant_id: string;
  merchant_category: string;
  timestamp: number;
  risk_score?: number;
  fraud_probability?: number;
};

export type TransactionsResponse = {
  transactions: TransactionRow[];
  count: number;
};

export type AlertRow = {
  id: number;
  transaction_id: string;
  risk_score: number;
  priority: string;
  status: string;
  created_at: number;
};

export type AlertsResponse = {
  alerts: AlertRow[];
  count: number;
};

export type MetricsResponse = {
  total_transactions: number;
  total_alerts: number;
  open_alerts: number;
  timestamp: number;
};

async function authFetch(
  path: string,
  options: RequestInit = {},
): Promise<Response> {
  const token = getStoredToken();
  const headers = new Headers(options.headers);
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  if (!headers.has("Content-Type") && options.body) {
    headers.set("Content-Type", "application/json");
  }
  return fetch(`${getApiBase()}${path}`, { ...options, headers });
}

export async function loginRequest(
  email: string,
  password: string,
): Promise<LoginResponse> {
  const res = await fetch(`${getApiBase()}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `Login failed (${res.status})`);
  }
  return res.json() as Promise<LoginResponse>;
}

export async function fetchTransactions(limit = 50): Promise<TransactionsResponse> {
  const res = await authFetch(`/api/transactions?limit=${limit}`);
  if (res.status === 401) throw new Error("unauthorized");
  if (!res.ok) throw new Error("Failed to load transactions");
  return res.json() as Promise<TransactionsResponse>;
}

export async function fetchAlerts(
  status: string,
  limit = 50,
): Promise<AlertsResponse> {
  const q = new URLSearchParams({ status, limit: String(limit) });
  const res = await authFetch(`/api/alerts?${q}`);
  if (res.status === 401) throw new Error("unauthorized");
  if (!res.ok) throw new Error("Failed to load alerts");
  return res.json() as Promise<AlertsResponse>;
}

export async function fetchMetrics(): Promise<MetricsResponse> {
  const res = await authFetch("/api/metrics");
  if (res.status === 401) throw new Error("unauthorized");
  if (!res.ok) throw new Error("Failed to load metrics");
  return res.json() as Promise<MetricsResponse>;
}

export async function resolveAlert(id: number): Promise<void> {
  const res = await authFetch(`/api/alerts/${id}/resolve`, { method: "POST" });
  if (res.status === 401) throw new Error("unauthorized");
  if (!res.ok) throw new Error("Failed to resolve alert");
}
