import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { RunDrawer, type Run } from "./RunDrawer";

const baseRun: Run = {
  id: "01HZXYZULIDEXAMPLE0000",
  trigger: "manual",
  status: "success",
  startedAt: "2026-05-23T10:00:00Z",
  finishedAt: "2026-05-23T10:00:42Z",
  durationMs: 42000,
  total: 3,
  triggered: 2,
  skipped: 1,
  failed: 0,
  sources: [
    {
      sourceId: "rf-blog",
      name: "Recorded Future Blog",
      status: "success",
      newArticleCount: 4,
      iocCount: 18,
    },
    {
      sourceId: "kev",
      name: "CISA KEV",
      status: "skipped",
      skipReason: "interval_not_due",
    },
  ],
};

describe("RunDrawer", () => {
  it("renders summary, aggregate and per-source rows", () => {
    render(<RunDrawer run={baseRun} onClose={() => undefined} />);
    expect(screen.getByRole("dialog", { name: /Run detail/ })).toBeInTheDocument();
    expect(screen.getByText("Recorded Future Blog")).toBeInTheDocument();
    expect(screen.getByText("CISA KEV")).toBeInTheDocument();
    expect(screen.getByText(/skipped: interval_not_due/)).toBeInTheDocument();
  });

  it("calls onClose when Escape is pressed", () => {
    const onClose = vi.fn();
    render(<RunDrawer run={baseRun} onClose={onClose} />);
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });

  it("calls onClose when the close button is clicked", () => {
    const onClose = vi.fn();
    render(<RunDrawer run={baseRun} onClose={onClose} />);
    fireEvent.click(screen.getByRole("button", { name: /Close/i }));
    expect(onClose).toHaveBeenCalled();
  });

  it("renders the error block when errorMessage is present", () => {
    const run: Run = { ...baseRun, status: "failed", errorMessage: "boom" };
    render(<RunDrawer run={run} onClose={() => undefined} />);
    expect(screen.getByText("boom")).toBeInTheDocument();
  });
});
