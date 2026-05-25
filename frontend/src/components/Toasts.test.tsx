import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ToastsProvider, useToasts } from "./Toasts";

function Pusher() {
  const { push } = useToasts();
  return (
    <button type="button" onClick={() => push("hello world", { duration: 500 })}>
      push
    </button>
  );
}

describe("Toasts", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders pushed toasts and removes them after their duration", () => {
    render(
      <ToastsProvider>
        <Pusher />
      </ToastsProvider>,
    );

    fireEvent.click(screen.getByText("push"));
    expect(screen.getByText("hello world")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(800);
    });
    expect(screen.queryByText("hello world")).not.toBeInTheDocument();
  });

  it("throws when useToasts is used outside provider", () => {
    const Probe = () => {
      useToasts();
      return null;
    };
    expect(() => render(<Probe />)).toThrow(/ToastsProvider/);
  });
});
