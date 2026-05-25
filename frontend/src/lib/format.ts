// Display-layer helpers for IoCs. Defang is intentionally not stored on
// the server — the API returns canonical `Value` and the client decides
// how to render it (see CLAUDE.md §3 and spec §2.8).

const HASH_TYPES = new Set(["md5", "sha1", "sha256", "sha512"]);

export function defang(value: string, type?: string): string {
  if (!value) return value;
  let v = value
    .replace(/\bhttps:\/\//gi, "hxxps://")
    .replace(/\bhttp:\/\//gi, "hxxp://");
  if (type && HASH_TYPES.has(type)) return v;
  v = v.replace(/\./g, "[.]");
  if (type === "email") v = v.replace(/@/g, "[@]");
  return v;
}

// formatDate returns "yyyy-mm-dd hh:mm:ss" in the user's local zone.
// Falls back to "—" when the timestamp is the zero time.
export function formatDate(input?: string | Date | null): string {
  if (!input) return "—";
  const d = typeof input === "string" ? new Date(input) : input;
  if (isNaN(d.getTime()) || d.getFullYear() < 2000) return "—";
  const pad = (n: number) => n.toString().padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

// fmtRelative returns a short "Ns ago" / "Nm ago" / ... label. Inputs
// older than ~30 days fall back to the absolute yyyy-mm-dd prefix.
export function fmtRelative(input?: string | Date | null, now?: Date): string {
  if (!input) return "—";
  const d = typeof input === "string" ? new Date(input) : input;
  const t = d.getTime();
  if (isNaN(t) || d.getFullYear() < 2000) return "—";
  const ref = now ? now.getTime() : Date.now();
  const diff = Math.round((ref - t) / 1000);
  if (diff < 0) return "in the future";
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 86400 * 30) return `${Math.floor(diff / 86400)}d ago`;
  return formatDate(d).slice(0, 10);
}

// fmtDuration renders a millisecond duration as "1234ms" / "5.2s" /
// "2m 14s". Null / undefined become an em-dash.
export function fmtDuration(ms?: number | null): string {
  if (ms == null) return "—";
  if (ms < 1000) return `${ms}ms`;
  const s = ms / 1000;
  if (s < 60) return `${s.toFixed(s < 10 ? 1 : 0)}s`;
  const m = Math.floor(s / 60);
  const rs = Math.round(s - m * 60);
  return `${m}m ${rs}s`;
}

// fmtNum is a thin Intl.NumberFormat wrapper that handles null / undefined.
export function fmtNum(n?: number | null): string {
  if (n == null) return "—";
  return new Intl.NumberFormat().format(n);
}
