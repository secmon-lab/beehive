import { describe, expect, it } from "vitest";
import { defang, formatDate } from "./format";

describe("defang", () => {
  it("replaces dots with [.]", () => {
    expect(defang("evil.example.com")).toBe("evil[.]example[.]com");
  });
  it("replaces http(s) scheme", () => {
    expect(defang("http://evil.example")).toBe("hxxp://evil[.]example");
    expect(defang("https://evil.example")).toBe("hxxps://evil[.]example");
  });
});

describe("formatDate", () => {
  it("returns em-dash for empty input", () => {
    expect(formatDate(null)).toBe("—");
    expect(formatDate(undefined)).toBe("—");
    expect(formatDate("")).toBe("—");
  });
  it("returns em-dash for the Go zero time", () => {
    expect(formatDate("0001-01-01T00:00:00Z")).toBe("—");
  });
  it("formats a real timestamp", () => {
    const out = formatDate("2026-05-21T03:04:05Z");
    expect(out).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/);
  });
});
