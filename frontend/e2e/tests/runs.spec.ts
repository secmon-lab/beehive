import { expect, test } from "@playwright/test";

// The Runs page is currently a placeholder — we render the title, the
// subtitle and an empty-state message. The smoke test just confirms the
// route renders without throwing.

test("runs page renders the empty state", async ({ page }) => {
  await page.goto("/runs");
  await expect(page.getByRole("heading", { name: "Runs" })).toBeVisible();
  await expect(page.getByText("No runs yet.")).toBeVisible();
});
