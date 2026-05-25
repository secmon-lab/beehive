import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertCircle,
  AlertTriangle,
  Check,
  Clock,
  Database,
  FileText,
  Inbox,
  Play,
  RefreshCw,
  Rss,
  ShieldAlert,
  Zap,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Card, CardFoot, CardHead } from "../components/Card";
import { EmptyState } from "../components/EmptyState";
import { Spinner } from "../components/Spinner";
import { Stat } from "../components/Stat";
import { StatusDot } from "../components/StatusDot";
import { useToasts } from "../components/Toasts";
import { useT } from "../i18n";
import { ApiError, fetchJSON, mutateJSON } from "../lib/api";
import { fmtNum, fmtRelative, formatDate } from "../lib/format";

interface Source {
  id: string;
  name: string;
  kind: "blog" | "feed" | string;
  type: string;
  url?: string;
  interval: string;
  disabled?: boolean;
}

interface SourceState {
  enabledOverride?: "none" | "force_off" | "force_on";
  lastFetchedAt?: string;
  lastStatus?: string;
  lastError?: string;
  totalIocCount?: number;
}

interface SourceView {
  source: Source;
  state?: SourceState;
}

interface SourceListResponse {
  sources: SourceView[];
}

function parseIntervalMs(s?: string): number {
  if (!s) return 60 * 60 * 1000;
  const m = s.match(/^(\d+)([smhd])$/);
  if (!m) return 60 * 60 * 1000;
  const n = parseInt(m[1], 10);
  const unit: Record<string, number> = {
    s: 1000,
    m: 60_000,
    h: 3_600_000,
    d: 86_400_000,
  };
  return n * (unit[m[2]] ?? 0);
}

function isDue(view: SourceView, nowMs: number): boolean {
  const last = view.state?.lastFetchedAt;
  if (!last) return true;
  const elapsed = nowMs - new Date(last).getTime();
  return elapsed > parseIntervalMs(view.source.interval);
}

function isEnabled(view: SourceView): boolean {
  return !view.source.disabled && view.state?.enabledOverride !== "force_off";
}

export function SourcesPage() {
  const t = useT();
  const queryClient = useQueryClient();
  const { push: toast } = useToasts();

  const { data, isLoading, error } = useQuery<SourceListResponse>({
    queryKey: ["sources"],
    queryFn: () => fetchJSON<SourceListResponse>("/api/v1/sources"),
  });

  // Pull the IoC stats here as well so the "IoCs stored" tile matches
  // the IoCs page exactly (both come from metrics/ioc_counts). Source
  // state's TotalIoCCount counts every fetch occurrence, which double-
  // counts when multiple sources surface the same indicator and makes
  // the two pages disagree.
  const iocStats = useQuery<{ total: number }>({
    queryKey: ["iocs", "stats"],
    queryFn: () => fetchJSON<{ total: number }>("/api/v1/iocs/stats"),
    staleTime: 5 * 60_000,
    refetchOnWindowFocus: false,
  });

  const sources = data?.sources ?? [];
  const nowMs = Date.now();

  const stats = useMemo(() => {
    const enabled = sources.filter(isEnabled);
    const due = enabled.filter((v) => isDue(v, nowMs));
    const failed = sources.filter((v) => v.state?.lastStatus === "failed");
    return {
      total: sources.length,
      enabled: enabled.length,
      paused: sources.length - enabled.length,
      due: due.length,
      failed: failed.length,
    };
  }, [sources, nowMs]);
  const totalIoCs = iocStats.data?.total ?? 0;

  const fetchMutation = useMutation({
    mutationFn: async (view: SourceView) => {
      await mutateJSON(
        `/api/v1/sources/${encodeURIComponent(view.source.id)}/fetch`,
        { method: "POST" },
      );
      return view;
    },
    onSuccess: (view) => {
      toast(t("sources.toast.fetched", { name: view.source.name }), {
        icon: <Check size={14} />,
      });
      queryClient.invalidateQueries({ queryKey: ["sources"] });
      queryClient.invalidateQueries({ queryKey: ["runs"] });
      queryClient.invalidateQueries({ queryKey: ["iocs", "stats"] });
    },
    onError: (err, view) => {
      // eslint-disable-next-line no-console
      console.error("fetch source failed", err);
      toast(t("sources.toast.fetchFailed", { name: view.source.name }), {
        icon: <AlertCircle size={14} />,
      });
    },
  });

  interface AsyncFetchResponse {
    runId: string;
    mode?: string;
    inProgress?: boolean;
  }

  // pollingRunId tracks the run we are waiting on (either freshly
  // started or an existing in-progress run handed back by the dedupe
  // check). It is cleared when the run reaches a terminal state.
  const [pollingRunId, setPollingRunId] = useState<string | null>(null);

  const fetchAllMutation = useMutation({
    mutationFn: () =>
      mutateJSON<AsyncFetchResponse>("/api/v1/fetch?mode=async", { method: "POST" }),
    onSuccess: (resp) => {
      if (!resp?.runId) return;
      setPollingRunId(resp.runId);
      const runShort = resp.runId.slice(-12);
      toast(
        resp.inProgress
          ? t("sources.toast.fetchAllInProgress", { runId: runShort })
          : t("sources.toast.fetchAllStarted", { runId: runShort }),
        { icon: <Zap size={14} /> },
      );
    },
    onError: (err) => {
      // eslint-disable-next-line no-console
      console.error("fetch all failed", err);
      toast(t("sources.toast.fetchAllFailed"), {
        icon: <AlertCircle size={14} />,
      });
    },
  });

  interface RunPollResponse {
    id: string;
    status?: string;
    triggered?: number;
    skipped?: number;
    failed?: number;
    errorMessage?: string;
  }

  const runPoll = useQuery<RunPollResponse>({
    queryKey: ["runs", "poll", pollingRunId],
    queryFn: () => fetchJSON<RunPollResponse>(`/api/v1/runs/${pollingRunId}`),
    enabled: !!pollingRunId,
    refetchInterval: (query) => {
      const s = query.state.data?.status;
      if (s === "success" || s === "failed" || s === "partial") return false;
      return 2000;
    },
  });

  useEffect(() => {
    const r = runPoll.data;
    if (!r) return;
    if (r.status !== "success" && r.status !== "failed" && r.status !== "partial") {
      return;
    }
    // Terminal state — surface a toast, invalidate dependent caches,
    // then stop polling.
    if (r.status === "failed") {
      toast(t("sources.toast.fetchAllFailed"), { icon: <AlertCircle size={14} /> });
    } else {
      toast(
        t("sources.toast.fetchedAll", { triggered: r.triggered ?? 0 }),
        { icon: <Check size={14} /> },
      );
    }
    setPollingRunId(null);
    queryClient.invalidateQueries({ queryKey: ["sources"] });
    queryClient.invalidateQueries({ queryKey: ["runs"] });
    queryClient.invalidateQueries({ queryKey: ["iocs", "stats"] });
    queryClient.invalidateQueries({ queryKey: ["iocs", "recent"] });
  }, [runPoll.data, queryClient, t, toast]);

  const isFetchAllBusy = fetchAllMutation.isPending || pollingRunId !== null;

  const onReload = () => {
    queryClient.invalidateQueries({ queryKey: ["sources"] });
    toast(t("sources.toast.reloaded"), { icon: <RefreshCw size={14} /> });
  };

  const showSpinner = isLoading;

  return (
    <div>
      <div className="page-head">
        <div className="page-head-text">
          <h1>{t("sources.title")}</h1>
          <p>{t("sources.subtitle")}</p>
        </div>
        <div className="page-head-actions">
          <button className="btn" type="button" onClick={onReload}>
            <RefreshCw size={13} />
            {t("sources.action.reload")}
          </button>
          <button
            className="btn btn-primary"
            type="button"
            onClick={() => fetchAllMutation.mutate()}
            disabled={isFetchAllBusy}
          >
            {isFetchAllBusy ? <Spinner /> : <Zap size={13} />}
            {isFetchAllBusy
              ? t("sources.action.fetchingAll")
              : `${t("sources.action.fetchAll")} (${stats.due})`}
          </button>
        </div>
      </div>

      <div className="stats">
        <Stat
          icon={<Database />}
          label={t("sources.stats.configured")}
          value={stats.total}
          meta={t("sources.stats.configuredMeta", {
            enabled: stats.enabled,
            paused: stats.paused,
          })}
        />
        <Stat
          icon={<Clock />}
          label={t("sources.stats.due")}
          value={stats.due}
          meta={t("sources.stats.dueMeta")}
        />
        <Stat
          icon={<AlertTriangle />}
          label={t("sources.stats.failing")}
          value={stats.failed}
          meta={t("sources.stats.failingMeta")}
          tone={stats.failed ? "danger" : "default"}
        />
        <Stat
          icon={<ShieldAlert />}
          label={t("sources.stats.iocs")}
          value={iocStats.isLoading ? <Spinner /> : fmtNum(totalIoCs)}
          meta={t("sources.stats.iocsMeta")}
        />
      </div>

      {error && (
        <div className="error-banner" role="alert">
          <AlertCircle size={14} />
          {t("sources.error.list")}: {(error as Error).message}
        </div>
      )}

      <Card>
        <CardHead
          icon={<Database size={14} />}
          title={t("sources.card.title")}
          count={sources.length}
          right={
            <span className="muted" style={{ fontSize: "var(--text-xs)" }}>
              {t("sources.card.sortedByName")}
            </span>
          }
        />
        {showSpinner ? (
          <div className="tbl-empty">
            <Spinner />
          </div>
        ) : sources.length === 0 ? (
          <EmptyState
            icon={<Inbox />}
            title={t("sources.empty.title")}
            description={t("sources.empty.description")}
          />
        ) : (
          <div className="tbl-wrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>{t("sources.column.name")}</th>
                  <th>{t("sources.column.kind")}</th>
                  <th>{t("sources.column.type")}</th>
                  <th>{t("sources.column.interval")}</th>
                  <th>{t("sources.column.override")}</th>
                  <th>{t("sources.column.lastStatus")}</th>
                  <th>{t("sources.column.lastFetchedAt")}</th>
                  <th style={{ textAlign: "right" }}> </th>
                </tr>
              </thead>
              <tbody>
                {sources.map((v) => {
                  const disabled =
                    v.source.disabled || v.state?.enabledOverride === "force_off";
                  const isFetching =
                    fetchMutation.isPending &&
                    fetchMutation.variables?.source.id === v.source.id;
                  const override = v.state?.enabledOverride;
                  return (
                    <tr key={v.source.id}>
                      <td>
                        <div className="stack">
                          <span
                            style={{
                              fontWeight: 500,
                              opacity: disabled ? 0.55 : 1,
                            }}
                          >
                            {v.source.name}
                          </span>
                          {v.source.url && (
                            <span
                              className="secondary mono trunc trunc-lg"
                              title={v.source.url}
                            >
                              {v.source.url}
                            </span>
                          )}
                        </div>
                      </td>
                      <td>
                        <span className="badge">
                          {v.source.kind === "feed" ? (
                            <Rss size={11} />
                          ) : (
                            <FileText size={11} />
                          )}
                          {v.source.kind}
                        </span>
                      </td>
                      <td className="mono muted">{v.source.type}</td>
                      <td className="mono">{v.source.interval}</td>
                      <td>
                        {!override || override === "none" ? (
                          <span className="muted">{t("sources.override.none")}</span>
                        ) : override === "force_off" ? (
                          <span className="badge badge-danger">
                            {t("sources.override.forceOff")}
                          </span>
                        ) : (
                          <span className="badge badge-success">
                            {t("sources.override.forceOn")}
                          </span>
                        )}
                      </td>
                      <td>
                        {v.state?.lastStatus ? (
                          <StatusDot status={v.state.lastStatus} />
                        ) : (
                          <span className="muted">{t("common.dash")}</span>
                        )}
                        {v.state?.lastError && (
                          <div
                            className="secondary trunc"
                            title={v.state.lastError}
                            style={{ color: "var(--color-danger)" }}
                          >
                            {v.state.lastError}
                          </div>
                        )}
                      </td>
                      <td className="mono muted" style={{ whiteSpace: "nowrap" }}>
                        {v.state?.lastFetchedAt ? (
                          <div className="stack">
                            <span style={{ color: "var(--color-text)" }}>
                              {formatDate(v.state.lastFetchedAt)}
                            </span>
                            <span className="secondary">
                              {fmtRelative(v.state.lastFetchedAt)}
                            </span>
                          </div>
                        ) : (
                          <span>{t("common.dash")}</span>
                        )}
                      </td>
                      <td className="actions">
                        <div className="row-actions">
                          <button
                            type="button"
                            className="btn btn-sm"
                            disabled={disabled || isFetching}
                            aria-label={`${t("sources.action.fetch")}: ${v.source.name}`}
                            onClick={() => fetchMutation.mutate(v)}
                          >
                            {isFetching ? <Spinner /> : <Play size={11} />}
                            {isFetching
                              ? t("sources.action.fetching")
                              : t("sources.action.fetch")}
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
        <CardFoot
          left={
            <>
              {sources.length} source{sources.length === 1 ? "" : "s"} ·{" "}
              {t("sources.foot.suffix")}{" "}
              <code>{t("sources.code.configPath")}</code>
            </>
          }
          right={
            <>
              <span className="kbd">{t("sources.foot.apiVerb")}</span>{" "}
              <code>{t("sources.foot.apiPath")}</code>
            </>
          }
        />
      </Card>
    </div>
  );
}

// Surface ApiError for any caller that re-throws it (not used directly
// today but kept exported so future tests can match on it).
export { ApiError };
