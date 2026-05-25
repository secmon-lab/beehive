import { AlertCircle, Clock, History, Play, X } from "lucide-react";
import { useEffect } from "react";
import { useT } from "../i18n";
import { fmtDuration, formatDate } from "../lib/format";
import { StatusDot } from "./StatusDot";

export interface RunSource {
  sourceId?: string;
  sourceKind?: string;
  status?: string;
  skipReason?: string;
  startedAt?: string;
  finishedAt?: string;
  articleCount?: number;
  newArticleCount?: number;
  iocCount?: number;
  errorMessage?: string;
  name?: string;
}

export interface Run {
  id: string;
  trigger?: string;
  status?: string;
  startedAt?: string;
  finishedAt?: string;
  durationMs?: number;
  total?: number;
  triggered?: number;
  skipped?: number;
  failed?: number;
  errorMessage?: string;
  sources?: RunSource[];
}

interface RunDrawerProps {
  run: Run;
  onClose: () => void;
}

export function RunDrawer({ run, onClose }: RunDrawerProps) {
  const t = useT();

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [onClose]);

  const sources = run.sources ?? [];

  return (
    <>
      <div className="drawer-scrim" onClick={onClose} />
      <aside className="drawer" role="dialog" aria-label={t("runs.detail.title")}>
        <div className="drawer-head">
          <div className="title">
            <History size={18} />
            {t("runs.detail.title")}
          </div>
          <button
            type="button"
            className="btn btn-icon btn-ghost"
            onClick={onClose}
            aria-label={t("common.close")}
          >
            <X size={14} />
          </button>
        </div>
        <div className="drawer-body">
          <div className="drawer-section">
            <h4>{t("runs.detail.summary")}</h4>
            <dl className="deflist">
              <dt>{t("runs.detail.dt.id")}</dt>
              <dd>
                <code>{run.id}</code>
              </dd>
              <dt>{t("runs.detail.dt.trigger")}</dt>
              <dd>
                {run.trigger === "scheduler" ? (
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
              </dd>
              <dt>{t("runs.detail.dt.status")}</dt>
              <dd>
                <StatusDot status={run.status} />
              </dd>
              <dt>{t("runs.detail.dt.started")}</dt>
              <dd className="mono">{formatDate(run.startedAt)}</dd>
              <dt>{t("runs.detail.dt.finished")}</dt>
              <dd className="mono">{formatDate(run.finishedAt)}</dd>
              <dt>{t("runs.detail.dt.duration")}</dt>
              <dd className="mono">{fmtDuration(run.durationMs)}</dd>
            </dl>
          </div>

          <div className="drawer-section">
            <h4>{t("runs.detail.aggregate")}</h4>
            <div className="drawer-stats">
              <DrawerStat label={t("runs.detail.agg.total")} value={run.total ?? 0} />
              <DrawerStat
                label={t("runs.detail.agg.triggered")}
                value={run.triggered ?? 0}
                tone="success"
              />
              <DrawerStat
                label={t("runs.detail.agg.skipped")}
                value={run.skipped ?? 0}
                tone="muted"
              />
              <DrawerStat
                label={t("runs.detail.agg.failed")}
                value={run.failed ?? 0}
                tone={run.failed ? "danger" : "muted"}
              />
            </div>
          </div>

          {run.errorMessage && (
            <div className="drawer-section">
              <h4 style={{ color: "var(--color-danger)" }}>{t("runs.detail.error")}</h4>
              <pre className="drawer-error">{run.errorMessage}</pre>
            </div>
          )}

          <div className="drawer-section">
            <h4>
              {t("runs.detail.perSource")} ({sources.length})
            </h4>
            <table className="subtbl">
              <thead>
                <tr>
                  <th>{t("runs.expand.column.source")}</th>
                  <th>{t("runs.expand.column.status")}</th>
                  <th className="num">{t("runs.expand.column.newArticles")}</th>
                  <th className="num">{t("runs.expand.column.iocs")}</th>
                </tr>
              </thead>
              <tbody>
                {sources.map((s, i) => (
                  <tr key={i}>
                    <td>
                      <div className="stack">
                        <span>{s.name ?? s.sourceId ?? "—"}</span>
                        {s.errorMessage && (
                          <span className="secondary mono reason-error">
                            {s.errorMessage}
                          </span>
                        )}
                        {s.skipReason && (
                          <span className="secondary mono">
                            {t("runs.expand.skippedPrefix")}
                            {s.skipReason}
                          </span>
                        )}
                      </div>
                    </td>
                    <td>
                      <StatusDot status={s.status} />
                    </td>
                    <td className="num">{s.newArticleCount ?? 0}</td>
                    <td
                      className="num"
                      style={{ fontWeight: s.iocCount ? 500 : 400 }}
                    >
                      {s.iocCount ?? 0}
                    </td>
                  </tr>
                ))}
                {sources.length === 0 && (
                  <tr>
                    <td colSpan={4} className="muted" style={{ textAlign: "center" }}>
                      —
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {sources.length > 0 && (
            <div className="drawer-section">
              <h4>{t("runs.expand.breakdown")}</h4>
              <p
                className="muted"
                style={{ fontSize: "var(--text-sm)", margin: 0 }}
              >
                <AlertCircle
                  size={12}
                  style={{ verticalAlign: "-2px", marginRight: 4 }}
                />
                Detailed counts (articles / new / IoCs) are visible inline in the
                table.
              </p>
            </div>
          )}
        </div>
      </aside>
    </>
  );
}

interface DrawerStatProps {
  label: string;
  value: number | string;
  tone?: "default" | "success" | "danger" | "muted";
}

function DrawerStat({ label, value, tone = "default" }: DrawerStatProps) {
  return (
    <div className="drawer-stat">
      <div className="label">{label}</div>
      <div className={`value${tone !== "default" ? ` ${tone}` : ""}`}>{value}</div>
    </div>
  );
}
