import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  AlertCircle,
  ArrowUpRight,
  CheckCircle,
  ChevronRight,
  Clock,
  ExternalLink,
  History,
  Play,
  RefreshCw,
} from "lucide-react";
import { Fragment, useMemo, useState } from "react";
import { Card, CardFoot, CardHead } from "../components/Card";
import { EmptyState } from "../components/EmptyState";
import { RunDrawer, type Run, type RunSource } from "../components/RunDrawer";
import { SourceCounts } from "../components/SourceCounts";
import { Spinner } from "../components/Spinner";
import { Stat } from "../components/Stat";
import { StatusDot } from "../components/StatusDot";
import { useToasts } from "../components/Toasts";
import { useT } from "../i18n";
import { fetchJSON } from "../lib/api";
import { fmtDuration, fmtNum, fmtRelative, formatDate } from "../lib/format";

interface RunListResponse {
  runs: Run[];
}

export function RunsPage() {
  const t = useT();
  const queryClient = useQueryClient();
  const { push: toast } = useToasts();
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [drawerRun, setDrawerRun] = useState<Run | null>(null);

  const { data, isLoading, error } = useQuery<RunListResponse>({
    queryKey: ["runs"],
    queryFn: () => fetchJSON<RunListResponse>("/api/v1/runs"),
  });

  const runs = data?.runs ?? [];

  const stats = useMemo(() => {
    const succ = runs.filter((r) => r.status === "success").length;
    const failed = runs.filter((r) => r.status === "failed").length;
    const partial = runs.filter((r) => r.status === "partial").length;
    const avg = runs.length
      ? Math.round(
          runs.reduce((acc, r) => acc + (r.durationMs ?? 0), 0) / runs.length,
        )
      : 0;
    return {
      total: runs.length,
      succ,
      failed,
      partial,
      avg,
      last: runs[0],
    };
  }, [runs]);

  const onRefresh = () => {
    queryClient.invalidateQueries({ queryKey: ["runs"] });
    toast(t("runs.toast.refreshed"), { icon: <RefreshCw size={14} /> });
  };

  const toggle = (id: string) =>
    setExpandedId((curr) => (curr === id ? null : id));

  return (
    <div>
      <div className="page-head">
        <div className="page-head-text">
          <h1>{t("runs.title")}</h1>
          <p>{t("runs.subtitle")}</p>
        </div>
        <div className="page-head-actions">
          <button
            type="button"
            className="btn btn-ghost"
            onClick={onRefresh}
          >
            <RefreshCw size={13} />
            {t("runs.action.refresh")}
          </button>
        </div>
      </div>

      <div className="stats">
        <Stat
          icon={<History />}
          label={t("runs.stats.runs")}
          value={stats.total}
          meta={t("runs.stats.runsMeta")}
        />
        <Stat
          icon={<CheckCircle />}
          label={t("runs.stats.successful")}
          value={stats.succ}
          meta={t("runs.stats.successfulMeta", {
            partial: stats.partial,
            failed: stats.failed,
          })}
          tone={stats.succ ? "success" : "default"}
        />
        <Stat
          icon={<Clock />}
          label={t("runs.stats.avgDuration")}
          value={fmtDuration(stats.avg)}
          meta={t("runs.stats.avgDurationMeta", { n: stats.total })}
        />
        <Stat
          icon={<Activity />}
          label={t("runs.stats.lastRun")}
          value={stats.last ? <StatusDot status={stats.last.status} /> : t("runs.stats.lastRunNone")}
          meta={
            stats.last
              ? fmtRelative(stats.last.startedAt)
              : t("common.dash")
          }
        />
      </div>

      {error && (
        <div className="error-banner" role="alert">
          <AlertCircle size={14} />
          {t("runs.error.list")}: {(error as Error).message}
        </div>
      )}

      <Card>
        <CardHead
          icon={<History size={14} />}
          title={t("runs.card.title")}
          count={runs.length}
          right={
            <span className="muted" style={{ fontSize: "var(--text-xs)" }}>
              {t("runs.card.subtitle")}
            </span>
          }
        />

        {isLoading ? (
          <div className="tbl-empty">
            <Spinner />
          </div>
        ) : runs.length === 0 ? (
          <EmptyState
            icon={<History />}
            title={t("runs.empty.title")}
            description={t("runs.empty.description")}
          />
        ) : (
          <div className="tbl-wrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th style={{ width: 28 }}> </th>
                  <th>{t("runs.column.run")}</th>
                  <th>{t("runs.column.trigger")}</th>
                  <th>{t("runs.column.status")}</th>
                  <th>{t("runs.column.started")}</th>
                  <th>{t("runs.column.duration")}</th>
                  <th style={{ textAlign: "right" }}>{t("runs.column.sources")}</th>
                  <th> </th>
                </tr>
              </thead>
              <tbody>
                {runs.map((r) => {
                  const open = expandedId === r.id;
                  return (
                    <Fragment key={r.id}>
                      <tr
                        className={`clickable${open ? " expanded" : ""}`}
                        onClick={() => toggle(r.id)}
                      >
                        <td style={{ paddingRight: 0 }}>
                          <span className={`chev${open ? " open" : ""}`}>
                            <ChevronRight />
                          </span>
                        </td>
                        <td className="mono">
                          <div className="stack">
                            <span style={{ color: "var(--color-text)" }}>
                              {r.id.slice(-12)}
                            </span>
                            <span className="secondary">…{r.id.slice(0, 8)}</span>
                          </div>
                        </td>
                        <td>
                          {r.trigger === "scheduler" ? (
                            <span className="badge">
                              <Clock size={11} />
                              {t("runs.trigger.scheduler")}
                            </span>
                          ) : (
                            <span className="badge badge-primary">
                              <Play size={11} />
                              {t("runs.trigger.manual")}
                            </span>
                          )}
                        </td>
                        <td>
                          <StatusDot status={r.status} />
                        </td>
                        <td className="mono muted" style={{ whiteSpace: "nowrap" }}>
                          <div className="stack">
                            <span style={{ color: "var(--color-text)" }}>
                              {formatDate(r.startedAt)}
                            </span>
                            <span className="secondary">
                              {fmtRelative(r.startedAt)}
                            </span>
                          </div>
                        </td>
                        <td className="mono num">{fmtDuration(r.durationMs)}</td>
                        <td style={{ textAlign: "right" }}>
                          <SourceCounts
                            triggered={r.triggered ?? 0}
                            skipped={r.skipped ?? 0}
                            failed={r.failed ?? 0}
                            total={r.total ?? 0}
                          />
                        </td>
                        <td className="actions">
                          <div className="row-actions">
                            <button
                              type="button"
                              className="btn btn-sm btn-ghost"
                              aria-label={t("runs.action.openDetailAria")}
                              onClick={(e) => {
                                e.stopPropagation();
                                setDrawerRun(r);
                              }}
                            >
                              <ExternalLink size={11} />
                              {t("runs.action.openDetail")}
                            </button>
                          </div>
                        </td>
                      </tr>
                      {open && (
                        <tr className="expand-row">
                          <td colSpan={8}>
                            <ExpandedRun
                              run={r}
                              onOpenDetail={() => setDrawerRun(r)}
                            />
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
        <CardFoot
          left={
            <>
              showing {runs.length} run{runs.length === 1 ? "" : "s"} ·{" "}
              {t("runs.foot.suffix")}
            </>
          }
          right={
            <>
              <span className="kbd">{t("runs.foot.apiVerb")}</span>{" "}
              <code>{t("runs.foot.apiPath")}</code>
            </>
          }
        />
      </Card>

      {drawerRun && (
        <RunDrawer run={drawerRun} onClose={() => setDrawerRun(null)} />
      )}
    </div>
  );
}

function ExpandedRun({
  run,
  onOpenDetail,
}: {
  run: Run;
  onOpenDetail: () => void;
}) {
  const t = useT();
  const sources: RunSource[] = run.sources ?? [];
  const triggered = sources.filter((s) => s.status !== "skipped");
  const newArticles = triggered.reduce(
    (acc, s) => acc + (s.newArticleCount ?? 0),
    0,
  );
  const totalIocs = triggered.reduce((acc, s) => acc + (s.iocCount ?? 0), 0);

  return (
    <div className="expand-content">
      <div className="expand-head">
        <div className="expand-meta">
          <MetaBlock
            label={t("runs.expand.meta.id")}
            value={<code className="mono">{run.id}</code>}
          />
          <MetaBlock
            label={t("runs.expand.meta.finished")}
            value={formatDate(run.finishedAt)}
            mono
          />
          <MetaBlock
            label={t("runs.expand.meta.newArticles")}
            value={fmtNum(newArticles)}
          />
          <MetaBlock
            label={t("runs.expand.meta.newIocs")}
            value={fmtNum(totalIocs)}
          />
        </div>
        <button className="btn btn-sm" type="button" onClick={onOpenDetail}>
          {t("runs.action.openFullDetail")} <ArrowUpRight size={12} />
        </button>
      </div>

      {run.errorMessage && (
        <div className="expand-error" role="alert">
          <AlertCircle />
          <div>
            <strong>{t("runs.expand.runFailed")}:</strong>{" "}
            <code>{run.errorMessage}</code>
          </div>
        </div>
      )}

      <h4>{t("runs.expand.breakdown")}</h4>
      <table className="subtbl">
        <thead>
          <tr>
            <th>{t("runs.expand.column.source")}</th>
            <th>{t("runs.expand.column.status")}</th>
            <th>{t("runs.expand.column.reason")}</th>
            <th className="num">{t("runs.expand.column.articles")}</th>
            <th className="num">{t("runs.expand.column.new")}</th>
            <th className="num">{t("runs.expand.column.iocs")}</th>
          </tr>
        </thead>
        <tbody>
          {sources.map((s, i) => (
            <tr key={i}>
              <td>{s.name ?? s.sourceId ?? "—"}</td>
              <td>
                <StatusDot status={s.status} />
              </td>
              <td className="muted" style={{ fontSize: 12 }}>
                {s.errorMessage ? (
                  <span className="reason-error">{s.errorMessage}</span>
                ) : s.skipReason ? (
                  <code className="reason-code">{s.skipReason}</code>
                ) : (
                  <span className="dim">{t("common.dash")}</span>
                )}
              </td>
              <td className="num">{s.articleCount ?? 0}</td>
              <td
                className={`num${s.newArticleCount ? " num-success" : " num-subtle"}`}
              >
                {s.newArticleCount ?? 0}
              </td>
              <td
                className="num"
                style={{
                  fontWeight: s.iocCount ? 500 : 400,
                  color: s.iocCount ? "var(--color-text)" : "var(--color-text-subtle)",
                }}
              >
                {s.iocCount ?? 0}
              </td>
            </tr>
          ))}
          {sources.length === 0 && (
            <tr>
              <td colSpan={6} className="muted" style={{ textAlign: "center" }}>
                {t("common.dash")}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function MetaBlock({
  label,
  value,
  mono,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <div className="meta-block">
      <span className="label">{label}</span>
      <span className={`value${mono ? " mono" : ""}`}>{value}</span>
    </div>
  );
}
