import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Pager, pickVisible } from "./Pager";

describe("pickVisible", () => {
  it("returns every page when count is small", () => {
    expect(pickVisible(2, 5)).toEqual([1, 2, 3, 4, 5]);
  });

  it("shows the forward window when in the middle", () => {
    // page 10 of 20 → look back 2, look forward 5
    expect(pickVisible(10, 20)).toEqual([
      1,
      "...",
      8,
      9,
      10,
      11,
      12,
      13,
      14,
      15,
      "...",
      20,
    ]);
  });

  it("biases forward at the start", () => {
    expect(pickVisible(1, 20)).toEqual([1, 2, 3, 4, 5, 6, "...", 20]);
  });

  it("collapses to trailing window at the end", () => {
    expect(pickVisible(20, 20)).toEqual([1, "...", 18, 19, 20]);
  });

  it("still shows forward window past the apparent end", () => {
    // page 10 of 15 still surfaces 11..15
    const out = pickVisible(10, 15);
    expect(out).toEqual([1, "...", 8, 9, 10, 11, 12, 13, 14, 15]);
  });
});

describe("Pager", () => {
  it("renders nothing for a single page", () => {
    const { container } = render(<Pager page={1} pageCount={1} onChange={() => undefined} />);
    expect(container.querySelector(".pager")).toBeNull();
  });

  it("renders prev / next disabled at boundaries", () => {
    render(<Pager page={1} pageCount={5} onChange={() => undefined} />);
    expect(screen.getByRole("button", { name: "Prev" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Next" })).not.toBeDisabled();
  });

  it("fires onChange with the target page", () => {
    const onChange = vi.fn();
    render(<Pager page={2} pageCount={5} onChange={onChange} />);
    fireEvent.click(screen.getByRole("button", { name: "Go to page 4" }));
    expect(onChange).toHaveBeenCalledWith(4);
  });

  it("marks the current page with aria-current", () => {
    render(<Pager page={3} pageCount={5} onChange={() => undefined} />);
    const current = screen.getByRole("button", { name: "Go to page 3" });
    expect(current).toHaveAttribute("aria-current", "page");
  });
});
