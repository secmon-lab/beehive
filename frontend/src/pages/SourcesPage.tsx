import { useQuery } from "@tanstack/react-query";
import { fetchJSON } from "../lib/api";
import { formatDate } from "../lib/format";
import { useT } from "../i18n";

interface SourceView {
  source: {
    id: string;
    name: string;
    kind: string;
    type: string;
    url?: string;
    interval: string;
    disabled?: boolean;
  };
  state?: {
    enabledOverride?: string;
    lastFetchedAt?: string;
    lastStatus?: string;
  };
}

interface SourceListResponse {
  sources: SourceView[];
}

export function SourcesPage() {
  const t = useT();
  const { data, isLoading, error } = useQuery<SourceListResponse>({
    queryKey: ["sources"],
    queryFn: () => fetchJSON<SourceListResponse>("/api/v1/sources"),
  });

  return (
    <section>
      <h2 className="page-title">{t("sources.title")}</h2>
      <p className="page-subtitle">{t("sources.subtitle")}</p>

      {isLoading && <p>...</p>}
      {error && <p role="alert">{(error as Error).message}</p>}

      {data && data.sources.length === 0 && (
        <div className="empty-state">{t("sources.empty")}</div>
      )}

      {data && data.sources.length > 0 && (
        <table className="data-table">
          <thead>
            <tr>
              <th>{t("sources.column.name")}</th>
              <th>{t("sources.column.kind")}</th>
              <th>{t("sources.column.type")}</th>
              <th>{t("sources.column.interval")}</th>
              <th>{t("sources.column.disabled")}</th>
              <th>{t("sources.column.override")}</th>
              <th>{t("sources.column.lastStatus")}</th>
              <th>{t("sources.column.lastFetchedAt")}</th>
            </tr>
          </thead>
          <tbody>
            {data.sources.map((v) => (
              <tr key={v.source.id}>
                <td>{v.source.name}</td>
                <td>{v.source.kind}</td>
                <td>{v.source.type}</td>
                <td>{v.source.interval}</td>
                <td>{v.source.disabled ? "yes" : ""}</td>
                <td>{v.state?.enabledOverride ?? ""}</td>
                <td>{v.state?.lastStatus ?? ""}</td>
                <td>{formatDate(v.state?.lastFetchedAt)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
