import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import {
  Activity,
  AlertCircle,
  CheckCircle,
  Globe,
  HelpCircle,
  History,
  Inbox,
  Link as LinkIcon,
  Search,
  Terminal,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Card, CardFoot, CardHead } from "../components/Card";
import { CopyButton } from "../components/CopyButton";
import { EmptyState } from "../components/EmptyState";
import { Pager } from "../components/Pager";
import { Spinner } from "../components/Spinner";
import { Stat } from "../components/Stat";
import { TypeChip } from "../components/TypeChip";
import { useT } from "../i18n";
import { ApiError, fetchJSON } from "../lib/api";
import { defang, fmtNum, fmtRelative, formatDate } from "../lib/format";

interface IoC {
  id: string;
  type: string;
  value: string;
  raw?: string;
  firstSeenAt?: string;
  lastSeenAt?: string;
}

interface IoCListCursor {
  lastSeenAt: string;
  id: string;
}

interface IoCListResponse {
  iocs: IoC[];
  nextCursor?: IoCListCursor;
}

interface IoCStatsResponse {
  total: number;
  byType: Record<string, number>;
}

const IOC_TYPES = [
  "ipv4",
  "ipv6",
  "domain",
  "url",
  "md5",
  "sha1",
  "sha256",
  "sha512",
  "email",
] as const;

type LookupState =
  | { kind: "idle" }
  | { kind: "loading"; type: string; value: string }
  | { kind: "found"; data: IoC; type: string; value: string }
  | { kind: "notfound"; type: string; value: string }
  | { kind: "error"; message: string; type: string; value: string };

export function IocsPage() {
  const t = useT();
  const [type, setType] = useState<string>("domain");
  const [value, setValue] = useState<string>("");
  const [lookup, setLookup] = useState<LookupState>({ kind: "idle" });
  // React-state mirror of compositionstart/end. Combined with the
  // nativeEvent.isComposing check, this is the only race-free way to
  // suppress Enter while CJK input is being confirmed.
  const composingRef = useRef(false);

  const PAGE_SIZE = 50;

  // Cursor-based pagination via React Query infinite query. The server
  // returns up to PAGE_SIZE IoCs per chunk plus a nextCursor we feed
  // back verbatim — both lastSeenAt and id are required, see backend
  // commentary in pkg/repository/firestore/ioc.go::ListRecentIoCsAfter.
  const recent = useInfiniteQuery<IoCListResponse, Error>({
    queryKey: ["iocs", "recent"],
    queryFn: async ({ pageParam }) => {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE) });
      const cursor = pageParam as IoCListCursor | undefined;
      if (cursor) {
        params.set("after", cursor.lastSeenAt);
        params.set("afterId", cursor.id);
      }
      return fetchJSON<IoCListResponse>(`/api/v1/iocs?${params.toString()}`);
    },
    initialPageParam: undefined as IoCListCursor | undefined,
    getNextPageParam: (last) => last.nextCursor ?? undefined,
  });

  // IoC stats are served from the precomputed counter document
  // (metrics/ioc_counts) — one Firestore read per call regardless of
  // collection size. Cache aggressively all the same.
  const stats = useQuery<IoCStatsResponse>({
    queryKey: ["iocs", "stats"],
    queryFn: () => fetchJSON<IoCStatsResponse>("/api/v1/iocs/stats"),
    staleTime: 5 * 60_000,
    refetchOnWindowFocus: false,
  });

  const fetchedPages = recent.data?.pages ?? [];
  const iocs = fetchedPages.flatMap((p) => p.iocs);
  const byType = stats.data?.byType ?? {};
  const domainCount = byType["domain"] ?? 0;
  const urlCount = byType["url"] ?? 0;
  const ipCount = (byType["ipv4"] ?? 0) + (byType["ipv6"] ?? 0);
  const hashCount =
    (byType["md5"] ?? 0) +
    (byType["sha1"] ?? 0) +
    (byType["sha256"] ?? 0) +
    (byType["sha512"] ?? 0);

  const [page, setPage] = useState(1);
  // Total page count is driven by the counter doc — that way we can
  // surface "page 10 of 100" before the user has scrolled that deep.
  const totalIoCs = stats.data?.total ?? iocs.length;
  const pageCount = Math.max(1, Math.ceil(totalIoCs / PAGE_SIZE));
  // Auto-prefetch the next chunk when the visible page would otherwise
  // run out of rows. We tolerate `iocs.length` not yet covering the
  // target page during the in-flight fetch — pageRows just renders
  // empty until the request resolves.
  useEffect(() => {
    const needed = page * PAGE_SIZE;
    if (
      iocs.length < needed &&
      recent.hasNextPage &&
      !recent.isFetchingNextPage
    ) {
      void recent.fetchNextPage();
    }
  }, [page, iocs.length, recent]);
  // Snap back to the last available page when the data set actually
  // shrinks below the current page index (refetch returning fewer rows).
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [pageCount, page]);
  const safePage = Math.min(page, pageCount);
  const pageStart = (safePage - 1) * PAGE_SIZE;
  const pageRows = iocs.slice(pageStart, pageStart + PAGE_SIZE);
  const isPagePending =
    pageRows.length === 0 &&
    (recent.isFetching || recent.isFetchingNextPage);

  const submit = async (overrideType?: string, overrideValue?: string) => {
    const targetType = overrideType ?? type;
    const targetValue = (overrideValue ?? value).trim();
    if (!targetValue) return;
    setLookup({ kind: "loading", type: targetType, value: targetValue });
    try {
      const params = new URLSearchParams({ type: targetType, value: targetValue });
      const data = await fetchJSON<IoC>(
        `/api/v1/iocs/lookup?${params.toString()}`,
      );
      setLookup({ kind: "found", data, type: targetType, value: targetValue });
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setLookup({ kind: "notfound", type: targetType, value: targetValue });
        return;
      }
      const message = err instanceof Error ? err.message : String(err);
      setLookup({
        kind: "error",
        message,
        type: targetType,
        value: targetValue,
      });
    }
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter") return;
    if (e.nativeEvent.isComposing || composingRef.current) {
      e.preventDefault();
      e.stopPropagation();
      return;
    }
    submit();
  };

  const tryExample = (t2: string, v: string) => {
    setType(t2);
    setValue(v);
    submit(t2, v);
  };

  const lookupOnRow = (i: IoC) => {
    setType(i.type);
    setValue(i.value);
    submit(i.type, i.value);
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  return (
    <div>
      <div className="page-head">
        <div className="page-head-text">
          <h1>{t("iocs.title")}</h1>
          <p>{t("iocs.subtitle")}</p>
        </div>
      </div>

      <div className="stats">
        <Stat
          icon={<Globe />}
          label={t("iocs.stats.domains")}
          value={stats.isLoading ? <Spinner /> : fmtNum(domainCount)}
          meta={t("iocs.stats.domainsMeta")}
        />
        <Stat
          icon={<LinkIcon />}
          label={t("iocs.stats.urls")}
          value={stats.isLoading ? <Spinner /> : fmtNum(urlCount)}
          meta={t("iocs.stats.urlsMeta")}
        />
        <Stat
          icon={<Activity />}
          label={t("iocs.stats.ips")}
          value={stats.isLoading ? <Spinner /> : fmtNum(ipCount)}
          meta={t("iocs.stats.ipsMeta")}
        />
        <Stat
          icon={<Terminal />}
          label={t("iocs.stats.hashes")}
          value={stats.isLoading ? <Spinner /> : fmtNum(hashCount)}
          meta={t("iocs.stats.hashesMeta")}
        />
      </div>

      <div className="lookup" role="region" aria-label={t("iocs.lookup.heading")}>
        <h3>
          <Search size={14} />
          {t("iocs.lookup.heading")}
        </h3>
        <p className="lookup-help">
          <span className="muted">{t("iocs.lookup.help")}</span>
          <span className="muted">{t("iocs.lookup.tryLabel")}</span>
          <button
            type="button"
            className="btn btn-chip"
            onClick={() => tryExample("domain", "lemonade-paint-supply.com")}
          >
            <code>lemonade-paint-supply.com</code>
          </button>
          <button
            type="button"
            className="btn btn-chip"
            onClick={() => tryExample("ipv4", "185.220.101.42")}
          >
            <code>185.220.101.42</code>
          </button>
          <button
            type="button"
            className="btn btn-chip"
            onClick={() =>
              tryExample(
                "sha256",
                "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
              )
            }
          >
            <code>e3b0c44…b855</code>
          </button>
        </p>
        <form
          className="lookup-form"
          onSubmit={(e) => {
            e.preventDefault();
            submit();
          }}
        >
          <select
            className="select"
            value={type}
            onChange={(e) => setType(e.target.value)}
            aria-label="IoC type"
          >
            {IOC_TYPES.map((opt) => (
              <option key={opt} value={opt}>
                {opt}
              </option>
            ))}
          </select>
          <input
            type="text"
            className="input mono"
            value={value}
            placeholder={t("iocs.placeholder.value")}
            spellCheck={false}
            autoCapitalize="off"
            autoCorrect="off"
            aria-label="IoC value"
            onChange={(e) => setValue(e.target.value)}
            onCompositionStart={() => {
              composingRef.current = true;
            }}
            onCompositionEnd={() => {
              composingRef.current = false;
            }}
            onKeyDown={onKeyDown}
          />
          <button
            type="submit"
            className="btn btn-primary"
            disabled={!value.trim() || lookup.kind === "loading"}
          >
            {lookup.kind === "loading" ? <Spinner /> : <Search size={13} />}
            {t("iocs.action.lookup")}
          </button>
        </form>

        <LookupResult lookup={lookup} />
      </div>

      <Card>
        <CardHead
          icon={<History size={14} />}
          title={t("iocs.recent.heading")}
          count={iocs.length}
          right={
            <span className="muted" style={{ fontSize: "var(--text-xs)" }}>
              {t("iocs.recent.cardMeta")}
            </span>
          }
        />

        {recent.error && (
          <div className="error-banner" role="alert" style={{ margin: "12px" }}>
            <AlertCircle size={14} />
            {t("iocs.error.list")}: {(recent.error as Error).message}
          </div>
        )}

        {recent.isLoading || isPagePending ? (
          <div className="tbl-empty">
            <Spinner />
          </div>
        ) : iocs.length === 0 ? (
          <EmptyState
            icon={<Inbox />}
            title={t("iocs.recent.empty.title")}
            description={t("iocs.recent.empty.description")}
          />
        ) : (
          <>
            <div className="tbl-wrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>{t("iocs.column.type")}</th>
                    <th>{t("iocs.column.value")}</th>
                    <th>{t("iocs.column.firstSeenAt")}</th>
                    <th>{t("iocs.column.lastSeenAt")}</th>
                    <th> </th>
                  </tr>
                </thead>
                <tbody>
                  {pageRows.map((i) => (
                    <tr key={i.id}>
                      <td>
                        <TypeChip type={i.type} />
                      </td>
                      <td className="mono">
                        <span className="trunc trunc-lg" title={i.value}>
                          {defang(i.value, i.type)}
                        </span>
                      </td>
                      <td className="mono muted" style={{ whiteSpace: "nowrap" }}>
                        {formatDate(i.firstSeenAt)}
                      </td>
                      <td className="mono muted" style={{ whiteSpace: "nowrap" }}>
                        <div className="stack">
                          <span style={{ color: "var(--color-text)" }}>
                            {formatDate(i.lastSeenAt)}
                          </span>
                          <span className="secondary">
                            {fmtRelative(i.lastSeenAt)}
                          </span>
                        </div>
                      </td>
                      <td className="actions">
                        <div className="row-actions">
                          <button
                            type="button"
                            className="btn btn-sm btn-ghost"
                            onClick={() => lookupOnRow(i)}
                            aria-label={t("iocs.recent.lookupAria")}
                            title={t("iocs.recent.lookupAria")}
                          >
                            <Search size={11} />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pager page={safePage} pageCount={pageCount} onChange={setPage} />
          </>
        )}
        <CardFoot
          left={
            <>
              {t("iocs.recent.total", { n: totalIoCs })} ·{" "}
              {t("iocs.foot.cardinality")}{" "}
              <code>{t("iocs.foot.cardinalityOrder")}</code>
            </>
          }
          right={
            <>
              <span className="kbd">{t("iocs.foot.apiVerb")}</span>{" "}
              <code>{t("iocs.foot.apiPath")}</code>
            </>
          }
        />
      </Card>
    </div>
  );
}

function LookupResult({ lookup }: { lookup: LookupState }) {
  const t = useT();

  if (lookup.kind === "idle") return null;

  if (lookup.kind === "loading") {
    return (
      <div className="lookup-result" role="status" aria-live="polite">
        <div className="res-head">
          <Spinner />
          {t("iocs.result.loading", { value: lookup.value })}
        </div>
      </div>
    );
  }

  if (lookup.kind === "error") {
    return (
      <div className="lookup-result error" role="alert">
        <div className="res-head">
          <AlertCircle />
          {t("iocs.result.error")}
        </div>
        <p style={{ color: "var(--color-danger)" }}>{lookup.message}</p>
      </div>
    );
  }

  if (lookup.kind === "notfound") {
    return (
      <div className="lookup-result notfound" role="status">
        <div className="res-head">
          <HelpCircle />
          {t("iocs.result.notFound")}
        </div>
        <p className="muted">
          {t("iocs.result.notFoundDescription", {
            type: lookup.type,
            value: defang(lookup.value, lookup.type),
          })}
        </p>
      </div>
    );
  }

  const d = lookup.data;
  return (
    <div className="lookup-result found">
      <div className="res-head">
        <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
          <CheckCircle />
          {t("iocs.result.found")}
        </span>
        <CopyButton value={d.value} label={t("iocs.action.copy")} />
      </div>
      <dl className="deflist">
        <dt>{t("iocs.result.dt.type")}</dt>
        <dd>
          <TypeChip type={d.type} />
        </dd>
        <dt>{t("iocs.result.dt.value")}</dt>
        <dd>
          <code>{defang(d.value, d.type)}</code>
        </dd>
        <dt>{t("iocs.result.dt.raw")}</dt>
        <dd>{d.raw ? <code>{d.raw}</code> : <span className="muted">—</span>}</dd>
        <dt>{t("iocs.result.dt.firstSeen")}</dt>
        <dd className="mono">{formatDate(d.firstSeenAt)}</dd>
        <dt>{t("iocs.result.dt.lastSeen")}</dt>
        <dd className="mono">
          {formatDate(d.lastSeenAt)}{" "}
          <span className="muted" style={{ fontFamily: "var(--font-sans)" }}>
            ({fmtRelative(d.lastSeenAt)})
          </span>
        </dd>
      </dl>
    </div>
  );
}
