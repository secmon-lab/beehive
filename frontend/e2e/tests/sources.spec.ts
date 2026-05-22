import { expect, test } from "@playwright/test";

// The Sources page is a thin read-only table. The smoke test confirms
// the page mounts and the top-level structure is rendered. Empty states
// (no sources yet) are valid — we only assert the nav and the title.

test("sources page mounts", async ({ page }) => {
  await page.goto("/sources");
  await expect(page.getByRole("heading", { name: "Sources" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Sources" })).toBeVisible();
  await expect(page.getByRole("link", { name: "IoCs" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Runs" })).toBeVisible();
});

test("root redirects to /sources", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveURL(/\/sources$/);
});
