import { expect, test } from "@playwright/test";

// The Runs page renders the run-history card and (when there is no
// data) an empty state. The smoke test covers both the empty path and
// a stubbed-list path so we know the card renders rows.

const okHealth = JSON.stringify({ status: "ok", version: "0.1.0" });

test.beforeEach(async ({ page }) => {
  await page.route("**/api/v1/health", (route) =>
    route.fulfill({ status: 200, contentType: "application/json", body: okHealth }),
  );
  await page.route("**/api/v1/sources", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ sources: [] }),
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

test("runs page renders the empty state", async ({ page }) => {
  await page.route("**/api/v1/runs", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ runs: [] }),
    }),
  );

  await page.goto("/runs");
  await expect(page.getByRole("heading", { name: "Runs", level: 1 })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Run history" })).toBeVisible();
  await expect(page.getByText("No runs since cold start")).toBeVisible();
});

test("runs page renders rows when the API returns runs", async ({ page }) => {
  await page.route("**/api/v1/runs", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        runs: [
          {
            id: "01HZXYZRUNID00000000000000",
            trigger: "manual",
            status: "success",
            startedAt: "2026-05-22T10:00:00Z",
            finishedAt: "2026-05-22T10:00:30Z",
            durationMs: 30000,
            total: 3,
            triggered: 2,
            skipped: 1,
            failed: 0,
            sources: [],
          },
        ],
      }),
    }),
  );

  await page.goto("/runs");
  await expect(page.locator("table.tbl tbody tr").first()).toBeVisible();
  await expect(page.getByText(/manual/)).toBeVisible();
});
