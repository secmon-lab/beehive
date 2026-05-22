import { Navigate, Route, Routes, NavLink } from "react-router-dom";
import { SourcesPage } from "./pages/SourcesPage";
import { IocsPage } from "./pages/IocsPage";
import { RunsPage } from "./pages/RunsPage";
import { useT } from "./i18n";
import "./styles/global.css";

export default function App() {
  const t = useT();
  return (
    <div className="app-shell">
      <header className="app-header">
        <h1 className="app-title">beehive</h1>
        <nav className="app-nav">
          <NavLink to="/sources">{t("nav.sources")}</NavLink>
          <NavLink to="/iocs">{t("nav.iocs")}</NavLink>
          <NavLink to="/runs">{t("nav.runs")}</NavLink>
        </nav>
      </header>
      <main className="app-main">
        <Routes>
          <Route path="/" element={<Navigate to="/sources" replace />} />
          <Route path="/sources" element={<SourcesPage />} />
          <Route path="/iocs" element={<IocsPage />} />
          <Route path="/runs" element={<RunsPage />} />
        </Routes>
      </main>
    </div>
  );
}
