// Display-layer helpers for IoCs. Defang is intentionally not stored on
// the server — the API returns canonical `Value` and the client decides
// how to render it (see CLAUDE.md §3 and spec §2.8).

export function defang(value: string): string {
  return value
    .replace(/\./g, "[.]")
    .replace(/^http:/, "hxxp:")
    .replace(/^https:/, "hxxps:");
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
