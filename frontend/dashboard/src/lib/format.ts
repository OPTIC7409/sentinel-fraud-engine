export function formatTime(unix: number): string {
  return new Date(unix * 1000).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function formatMoney(amount: number, currency: string): string {
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency: currency || "USD",
      maximumFractionDigits: 2,
    }).format(amount);
  } catch {
    return `${amount.toFixed(2)} ${currency}`;
  }
}

export function riskTone(score: number | undefined): {
  label: string;
  className: string;
} {
  if (score == null) {
    return { label: "—", className: "text-zinc-500" };
  }
  if (score >= 75) {
    return { label: String(score), className: "text-risk-critical font-medium" };
  }
  if (score >= 50) {
    return { label: String(score), className: "text-risk-high font-medium" };
  }
  if (score >= 25) {
    return { label: String(score), className: "text-risk-mid" };
  }
  return { label: String(score), className: "text-risk-low" };
}
