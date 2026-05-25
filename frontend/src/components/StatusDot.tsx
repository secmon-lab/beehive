// StatusDot renders a small colored dot + optional pill for run/source
// status strings ("success" | "failed" | "running" | "skipped" | ...).
// The "showAs" prop chooses between the bare dot+label or a badge pill.

export type StatusKind =
  | "success"
  | "failed"
  | "running"
  | "skipped"
  | "partial"
  | "warning"
  | "pending"
  | string;

interface StatusDotProps {
  status?: StatusKind;
  label?: string;
  showAs?: "dot" | "pill";
}

const BADGE_VARIANT: Record<string, string> = {
  success: "badge-success",
  failed: "badge-danger",
  running: "badge-primary",
  partial: "badge-warning",
  warning: "badge-warning",
};

function titleCase(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

export function StatusDot({ status, label, showAs = "dot" }: StatusDotProps) {
  const s = status || "pending";
  const text = label ?? titleCase(s);

  if (showAs === "pill") {
    const variant = BADGE_VARIANT[s] || "";
    return (
      <span className={`badge ${variant}`.trim()}>
        <span
          className="dot"
          style={{ background: "currentColor", boxShadow: "none" }}
        />
        {text}
      </span>
    );
  }

  return (
    <span className={`status s-${s}`}>
      <span className="dot" />
      {text}
    </span>
  );
}
