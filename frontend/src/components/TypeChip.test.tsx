import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TypeChip } from "./TypeChip";

describe("TypeChip", () => {
  it("applies a per-type class", () => {
    const { container } = render(<TypeChip type="domain" />);
    expect(container.querySelector(".type-chip.type-chip-domain")).not.toBeNull();
  });

  it("renders the type text", () => {
    const { getByText } = render(<TypeChip type="sha256" />);
    expect(getByText("sha256")).toBeInTheDocument();
  });
});
