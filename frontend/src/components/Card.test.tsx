import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Card, CardFoot, CardHead } from "./Card";

describe("Card", () => {
  it("renders children inside a card class", () => {
    const { container } = render(<Card>hello</Card>);
    expect(container.querySelector(".card")).not.toBeNull();
    expect(screen.getByText("hello")).toBeInTheDocument();
  });

  it("merges extra class names", () => {
    const { container } = render(<Card className="extra">x</Card>);
    expect(container.querySelector(".card.extra")).not.toBeNull();
  });
});

describe("CardHead", () => {
  it("renders title and count", () => {
    render(<CardHead title="Sources" count={3} />);
    expect(screen.getByRole("heading", { name: /Sources/ })).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("omits count when undefined", () => {
    const { container } = render(<CardHead title="x" />);
    expect(container.querySelector(".count")).toBeNull();
  });

  it("renders the right slot", () => {
    render(<CardHead title="x" right={<span>extras</span>} />);
    expect(screen.getByText("extras")).toBeInTheDocument();
  });
});

describe("CardFoot", () => {
  it("renders both slots", () => {
    render(<CardFoot left="left" right="right" />);
    expect(screen.getByText("left")).toBeInTheDocument();
    expect(screen.getByText("right")).toBeInTheDocument();
  });
});
