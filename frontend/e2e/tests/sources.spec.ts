import { expect, test } from "@playwright/test";

// The Sources page is the default landing route. The smoke test confirms
// the page mounts and the top-level structure is rendered. Empty states
// (no sources yet) are valid — we only assert the nav and the title.

const emptySourceList = JSON.stringify({ sources: [] });
const okHealth = JSON.stringify({ status: "ok", version: "0.1.0" });

test.beforeEach(async ({ page }) => {
  await page.route("**/api/v1/health", (route) =>
    route.fulfill({ status: 200, contentType: "application/json", body: okHealth }),
  );
  await page.route("**/api/v1/sources", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: emptySourceList,
    }),
  );
  await page.route("**/api/v1/runs", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ runs: [] }),
    }),
  );
  await page.route("**/api/v1/iocs?**", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ iocs: [] }),
    }),
  );
});

test("sources page mounts", async ({ page }) => {
  await page.goto("/sources");
  await expect(page.getByRole("heading", { name: "Sources", level: 1 })).toBeVisible();
  await expect(page.getByRole("link", { name: /Sources/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /IoCs/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /Runs/ })).toBeVisible();
});

test("root redirects to /sources", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveURL(/\/sources$/);
});

test("brand mark + version pill render", async ({ page }) => {
  await page.goto("/sources");
  await expect(page.getByLabel("beehive home")).toBeVisible();
  await expect(page.getByText("v0.1.0")).toBeVisible();
});
