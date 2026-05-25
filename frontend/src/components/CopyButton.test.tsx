import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CopyButton } from "./CopyButton";

describe("CopyButton", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("writes to clipboard and flips the label to 'Copied'", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });

    render(<CopyButton value="example.com" />);
    const btn = screen.getByRole("button");
    expect(btn).toHaveTextContent("Copy");

    fireEvent.click(btn);
    expect(writeText).toHaveBeenCalledWith("example.com");
    await waitFor(() => expect(btn).toHaveTextContent("Copied"));
  });

  it("uses custom label and aria-label when provided", () => {
    render(<CopyButton value="x" label="Copy raw" ariaLabel="copy raw value" />);
    const btn = screen.getByRole("button", { name: "copy raw value" });
    expect(btn).toHaveTextContent("Copy raw");
  });
});
