import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ApiError, fetchJSON } from "../lib/api";
import { defang, formatDate } from "../lib/format";
import { useT } from "../i18n";

interface IoCResponse {
  id: string;
  type: string;
  value: string;
  raw?: string;
  firstSeenAt?: string;
  lastSeenAt?: string;
}

interface IoCListResponse {
  iocs: IoCResponse[];
}

const IOC_TYPES = [
  "ipv4", "ipv6", "domain", "url",
  "md5", "sha1", "sha256", "sha512", "email",
] as const;

export function IocsPage() {
  const t = useT();
  const [type, setType] = useState<string>("ipv4");
  const [value, setValue] = useState<string>("");
  const [result, setResult] = useState<IoCResponse | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [busy, setBusy] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const recent = useQuery<IoCListResponse>({
    queryKey: ["iocs", "recent"],
    queryFn: () => fetchJSON<IoCListResponse>("/api/v1/iocs?limit=100"),
  });

  async function lookup() {
    setBusy(true);
    setNotFound(false);
    setErrorMsg(null);
    setResult(null);
    try {
      const q = new URLSearchParams({ type, value });
      const res = await fetchJSON<IoCResponse>(`/api/v1/iocs/lookup?${q.toString()}`);
      setResult(res);
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setNotFound(true);
      } else {
        // eslint-disable-next-line no-console
        console.error(err);
        const msg = err instanceof Error ? err.message : String(err);
        setErrorMsg(msg);
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <section>
      <h2 className="page-title">{t("iocs.title")}</h2>
      <p className="page-subtitle">{t("iocs.subtitle")}</p>

      <form
        className="lookup-form"
        onSubmit={(e) => {
          e.preventDefault();
          lookup();
        }}
      >
        <select value={type} onChange={(e) => setType(e.target.value)}>
          {IOC_TYPES.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>
        <input
          aria-label="value"
          placeholder={t("iocs.placeholder.value")}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          // IME guard at the keystroke level: nativeEvent.isComposing is
          // the only race-free signal that an Enter is part of an IME
          // confirm. A React-state mirror of compositionstart/end can
          // stay sticky and silently block button clicks too.
          onKeyDown={(e) => {
            if (e.key === "Enter" && e.nativeEvent.isComposing) {
              e.preventDefault();
              e.stopPropagation();
            }
          }}
        />
        <button type="submit" disabled={busy || value === ""}>
          {t("iocs.action.lookup")}
        </button>
      </form>

      {busy && <p role="status">…</p>}
      {notFound && <p role="status">{t("iocs.notFound")}</p>}
      {errorMsg && (
        <p role="alert" className="error-message">
          {errorMsg}
        </p>
      )}

      {result && (
        <article className="result-card">
          <dl>
            <dt>type</dt>
            <dd>{result.type}</dd>
            <dt>value</dt>
            <dd>
              <code>{defang(result.value)}</code>
            </dd>
            <dt>raw</dt>
            <dd>{result.raw ? <code>{result.raw}</code> : "—"}</dd>
            <dt>first seen</dt>
            <dd>{formatDate(result.firstSeenAt)}</dd>
            <dt>last seen</dt>
            <dd>{formatDate(result.lastSeenAt)}</dd>
          </dl>
        </article>
      )}

      <h3 className="section-heading">{t("iocs.recent.heading")}</h3>
      {recent.isLoading && <p role="status">…</p>}
      {recent.error && (
        <p role="alert" className="error-message">
          {(recent.error as Error).message}
        </p>
      )}
      {recent.data && recent.data.iocs.length === 0 && (
        <div className="empty-state">{t("iocs.recent.empty")}</div>
      )}
      {recent.data && recent.data.iocs.length > 0 && (
        <table className="data-table">
          <thead>
            <tr>
              <th>{t("iocs.column.type")}</th>
              <th>{t("iocs.column.value")}</th>
              <th>{t("iocs.column.firstSeenAt")}</th>
              <th>{t("iocs.column.lastSeenAt")}</th>
            </tr>
          </thead>
          <tbody>
            {recent.data.iocs.map((i) => (
              <tr key={i.id}>
                <td>{i.type}</td>
                <td>
                  <code>{defang(i.value)}</code>
                </td>
                <td>{formatDate(i.firstSeenAt)}</td>
                <td>{formatDate(i.lastSeenAt)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
