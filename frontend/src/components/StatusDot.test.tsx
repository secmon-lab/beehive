import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StatusDot } from "./StatusDot";

describe("StatusDot", () => {
  it("applies the s-{status} class for known statuses", () => {
    const { container } = render(<StatusDot status="success" />);
    expect(container.querySelector(".status.s-success")).not.toBeNull();
  });

  it("falls back to s-pending when status is omitted", () => {
    const { container } = render(<StatusDot />);
    expect(container.querySelector(".status.s-pending")).not.toBeNull();
  });

  it("titlecases the status when no label is given", () => {
    const { getByText } = render(<StatusDot status="failed" />);
    expect(getByText("Failed")).toBeInTheDocument();
  });

  it("honours the explicit label", () => {
    const { getByText } = render(<StatusDot status="success" label="OK" />);
    expect(getByText("OK")).toBeInTheDocument();
  });

  it("renders as a pill when showAs='pill'", () => {
    const { container } = render(<StatusDot status="success" showAs="pill" />);
    expect(container.querySelector(".badge.badge-success")).not.toBeNull();
  });
});
