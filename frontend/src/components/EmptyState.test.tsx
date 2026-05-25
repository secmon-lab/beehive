import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { EmptyState } from "./EmptyState";

describe("EmptyState", () => {
  it("renders title, icon, description and action", () => {
    render(
      <EmptyState
        icon={<svg data-testid="icon" />}
        title="No data"
        description="Try again later."
        action={<button>Retry</button>}
      />,
    );
    expect(screen.getByText("No data")).toBeInTheDocument();
    expect(screen.getByText("Try again later.")).toBeInTheDocument();
    expect(screen.getByTestId("icon")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("omits description when not given", () => {
    const { container } = render(<EmptyState icon={<svg />} title="x" />);
    expect(container.querySelector(".empty p")).toBeNull();
  });
});
