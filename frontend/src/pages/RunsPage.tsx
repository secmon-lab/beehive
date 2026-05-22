import { useT } from "../i18n";

// The MVP backend exposes individual runs at /api/v1/runs/{id} but no
// list endpoint yet (history is BigQuery's job). The Runs page renders
// a brief explanation and a link to the BigQuery docs; richer history
// browsing is intentionally postponed.

export function RunsPage() {
  const t = useT();
  return (
    <section>
      <h2 className="page-title">{t("runs.title")}</h2>
      <p className="page-subtitle">{t("runs.subtitle")}</p>
      <div className="empty-state">{t("runs.empty")}</div>
    </section>
  );
}
