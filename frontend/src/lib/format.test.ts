import { describe, expect, it } from "vitest";
import { defang, fmtDuration, fmtNum, fmtRelative, formatDate } from "./format";

describe("defang", () => {
  it("replaces dots with [.]", () => {
    expect(defang("evil.example.com")).toBe("evil[.]example[.]com");
  });
  it("replaces http(s) scheme", () => {
    expect(defang("http://evil.example")).toBe("hxxp://evil[.]example");
    expect(defang("https://evil.example")).toBe("hxxps://evil[.]example");
  });
  it("skips dot replacement for hashes", () => {
    const h = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";
    expect(defang(h, "sha256")).toBe(h);
    expect(defang(h, "md5")).toBe(h);
    expect(defang(h, "sha1")).toBe(h);
    expect(defang(h, "sha512")).toBe(h);
  });
  it("replaces @ for emails", () => {
    expect(defang("attacker@evil.example", "email")).toBe("attacker[@]evil[.]example");
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

describe("fmtRelative", () => {
  const now = new Date("2026-05-23T12:00:00Z");
  it("returns em-dash for null", () => {
    expect(fmtRelative(null, now)).toBe("—");
    expect(fmtRelative(undefined, now)).toBe("—");
  });
  it("formats seconds, minutes, hours, days", () => {
    expect(fmtRelative(new Date(now.getTime() - 5_000), now)).toBe("5s ago");
    expect(fmtRelative(new Date(now.getTime() - 120_000), now)).toBe("2m ago");
    expect(fmtRelative(new Date(now.getTime() - 4 * 3600_000), now)).toBe("4h ago");
    expect(fmtRelative(new Date(now.getTime() - 3 * 86400_000), now)).toBe("3d ago");
  });
  it("flags future inputs", () => {
    expect(fmtRelative(new Date(now.getTime() + 60_000), now)).toBe("in the future");
  });
  it("falls back to absolute date past 30 days", () => {
    const old = new Date(now.getTime() - 60 * 86400_000);
    expect(fmtRelative(old, now)).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });
});

describe("fmtDuration", () => {
  it("returns em-dash for null", () => {
    expect(fmtDuration(null)).toBe("—");
    expect(fmtDuration(undefined)).toBe("—");
  });
  it("renders sub-second as ms", () => {
    expect(fmtDuration(0)).toBe("0ms");
    expect(fmtDuration(750)).toBe("750ms");
  });
  it("renders seconds with one decimal under 10s", () => {
    expect(fmtDuration(5_200)).toBe("5.2s");
  });
  it("renders integer seconds when >= 10s", () => {
    expect(fmtDuration(45_000)).toBe("45s");
  });
  it("renders minutes + seconds past 60s", () => {
    expect(fmtDuration(134_000)).toBe("2m 14s");
  });
});

describe("fmtNum", () => {
  it("returns em-dash for null", () => {
    expect(fmtNum(null)).toBe("—");
    expect(fmtNum(undefined)).toBe("—");
  });
  it("inserts thousand separators", () => {
    expect(fmtNum(1234567)).toMatch(/1.234.567|1,234,567/);
  });
});
