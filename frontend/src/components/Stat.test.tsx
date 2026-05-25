import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Stat } from "./Stat";

describe("Stat", () => {
  it("renders label, value and meta", () => {
    render(<Stat label="Configured" value={5} meta="2 enabled" />);
    expect(screen.getByText("Configured")).toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
    expect(screen.getByText("2 enabled")).toBeInTheDocument();
  });

  it("applies tone class when not default", () => {
    const { container } = render(<Stat label="Failing" value={1} tone="danger" />);
    expect(container.querySelector(".stat-value.danger")).not.toBeNull();
  });

  it("omits meta when not provided", () => {
    const { container } = render(<Stat label="x" value="y" />);
    expect(container.querySelector(".stat-meta")).toBeNull();
  });
});
