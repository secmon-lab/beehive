import { describe, expect, it } from "vitest";
import { t } from "./index";

describe("i18n", () => {
  it("returns the English string for known keys", () => {
    expect(t("nav.sources")).toBe("Sources");
    expect(t("iocs.title")).toBe("IoC lookup");
  });
});
