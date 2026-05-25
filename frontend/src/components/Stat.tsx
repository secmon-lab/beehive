import type { ReactNode } from "react";

interface StatProps {
  icon?: ReactNode;
  label: string;
  value: ReactNode;
  meta?: ReactNode;
  tone?: "default" | "success" | "danger";
}

export function Stat({ icon, label, value, meta, tone = "default" }: StatProps) {
  return (
    <div className="stat">
      <div className="stat-label">
        {icon}
        {label}
      </div>
      <div className={`stat-value${tone !== "default" ? ` ${tone}` : ""}`}>{value}</div>
      {meta !== undefined && <div className="stat-meta">{meta}</div>}
    </div>
  );
}
