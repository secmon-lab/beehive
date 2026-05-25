import { describe, expect, it } from "vitest";
import { en } from "./en";
import { t } from "./index";

describe("i18n", () => {
  it("returns the English string for known keys", () => {
    expect(t("nav.sources")).toBe("Sources");
    expect(t("iocs.title")).toBe("IoCs");
    expect(t("runs.title")).toBe("Runs");
  });

  it("returns a non-empty string for every dictionary key", () => {
    for (const key of Object.keys(en) as Array<keyof typeof en>) {
      const value = t(key);
      expect(value, `key ${key} should be a non-empty string`).toBeTruthy();
      expect(typeof value).toBe("string");
    }
  });
});
