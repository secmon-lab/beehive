import { ChevronLeft, ChevronRight } from "lucide-react";
import { useT } from "../i18n";

interface PagerProps {
  page: number;
  pageCount: number;
  onChange: (page: number) => void;
}

// Pager renders a compact prev / page-numbers / next strip. It clamps
// the visible page list to ~7 buttons with ellipses when there are
// more pages than fit; for the IoCs page we cap pageCount at 10 so the
// strip always renders without ellipsis in practice.
export function Pager({ page, pageCount, onChange }: PagerProps) {
  const t = useT();
  if (pageCount <= 1) return null;

  const visible = pickVisible(page, pageCount);

  return (
    <nav className="pager" aria-label="Pagination">
      <button
        type="button"
        className="btn btn-sm btn-ghost"
        onClick={() => onChange(page - 1)}
        disabled={page <= 1}
        aria-label={t("common.pagination.prev")}
      >
        <ChevronLeft size={13} />
        {t("common.pagination.prev")}
      </button>

      <ul className="pager-pages">
        {visible.map((p, idx) =>
          p === "..." ? (
            <li key={`gap-${idx}`} className="pager-gap" aria-hidden="true">
              …
            </li>
          ) : (
            <li key={p}>
              <button
                type="button"
                className={`pager-num${p === page ? " active" : ""}`}
                onClick={() => onChange(p)}
                aria-label={t("common.pagination.goto", { n: p })}
                aria-current={p === page ? "page" : undefined}
              >
                {p}
              </button>
            </li>
          ),
        )}
      </ul>

      <button
        type="button"
        className="btn btn-sm btn-ghost"
        onClick={() => onChange(page + 1)}
        disabled={page >= pageCount}
        aria-label={t("common.pagination.next")}
      >
        {t("common.pagination.next")}
        <ChevronRight size={13} />
      </button>
    </nav>
  );
}

// pickVisible chooses which page numbers to surface in the pager.
// The window is intentionally asymmetric — two slots looking back and
// five slots looking forward — so a user landing on page 10 still sees
// 11..15 ahead, not just 8..12. First and last pages are always
// visible; any gap between exposed numbers collapses into a "..."
// marker.
//
//   pickVisible(10, 20) → [1, "...", 8, 9, 10, 11, 12, 13, 14, 15, "...", 20]
//   pickVisible(1, 20)  → [1, 2, 3, 4, 5, 6, "...", 20]
//   pickVisible(20, 20) → [1, "...", 18, 19, 20]
//
// Exported for unit tests.
export function pickVisible(page: number, pageCount: number): Array<number | "..."> {
  if (pageCount <= 1) return [];
  const out: Array<number | "..."> = [];
  if (pageCount <= 8) {
    for (let i = 1; i <= pageCount; i++) out.push(i);
    return out;
  }
  const window = new Set<number>([1, pageCount]);
  for (let i = page - 2; i <= page + 5; i++) {
    if (i >= 1 && i <= pageCount) window.add(i);
  }
  const sorted = [...window].sort((a, b) => a - b);
  for (let i = 0; i < sorted.length; i++) {
    if (i > 0 && sorted[i] - sorted[i - 1] > 1) out.push("...");
    out.push(sorted[i]);
  }
  return out;
}
