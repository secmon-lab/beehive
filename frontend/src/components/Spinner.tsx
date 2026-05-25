import { useT } from "../i18n";

export function Spinner() {
  const t = useT();
  return <span className="spinner" role="status" aria-label={t("common.loading")} />;
}
