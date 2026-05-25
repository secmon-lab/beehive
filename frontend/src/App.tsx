import { useQuery } from "@tanstack/react-query";
import { Database, Hexagon, History, ShieldAlert } from "lucide-react";
import { Navigate, NavLink, Route, Routes } from "react-router-dom";
import { ToastsProvider } from "./components/Toasts";
import { useT } from "./i18n";
import { fetchJSON } from "./lib/api";
import { IocsPage } from "./pages/IocsPage";
import { RunsPage } from "./pages/RunsPage";
import { SourcesPage } from "./pages/SourcesPage";
import "./styles/global.css";

interface HealthResponse {
  status: string;
  version?: string;
}

export default function App() {
  const t = useT();
  const health = useQuery<HealthResponse>({
    queryKey: ["health"],
    queryFn: () => fetchJSON<HealthResponse>("/api/v1/health"),
    // The version tag in the header is best-effort — failure should not
    // cascade into the rest of the UI, so retry stays off and stale time
    // is long.
    retry: false,
    staleTime: 5 * 60_000,
  });

  return (
    <ToastsProvider>
      <div className="app">
        <header className="app-header">
          <a href="/" className="brand" aria-label={t("header.brand.aria")}>
            <span className="brand-mark" aria-hidden="true">
              <Hexagon size={14} />
            </span>
            <span className="brand-name">
              beehive
              {health.data?.version && (
                <span className="brand-tag">v{health.data.version}</span>
              )}
            </span>
          </a>

          <nav className="nav" aria-label="Primary">
            <NavLink to="/sources">
              <Database className="nav-icon" />
              {t("nav.sources")}
            </NavLink>
            <NavLink to="/iocs">
              <ShieldAlert className="nav-icon" />
              {t("nav.iocs")}
            </NavLink>
            <NavLink to="/runs">
              <History className="nav-icon" />
              {t("nav.runs")}
            </NavLink>
          </nav>

          <div className="header-right">
            <span className="env-pill" title="Cloud Run status">
              <span className="env-dot" />
              {t("header.env.prefix")}
              {health.data?.status ? ` · ${health.data.status}` : ""}
            </span>
          </div>
        </header>

        <main className="main">
          <Routes>
            <Route path="/" element={<Navigate to="/sources" replace />} />
            <Route path="/sources" element={<SourcesPage />} />
            <Route path="/iocs" element={<IocsPage />} />
            <Route path="/runs" element={<RunsPage />} />
          </Routes>
        </main>
      </div>
    </ToastsProvider>
  );
}
