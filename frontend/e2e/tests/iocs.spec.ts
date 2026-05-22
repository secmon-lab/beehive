import { expect, test } from "@playwright/test";

// IoC page is the only interactive page. We stub both the lookup
// endpoint (per-test specific) and the list endpoint (kept generic
// across all specs) so the suite stays offline from a real Firestore.

const emptyList = JSON.stringify({ iocs: [] });

test("ioc lookup shows not found", async ({ page }) => {
  await page.route("**/api/v1/iocs?**", (route) =>
    route.fulfill({ status: 200, contentType: "application/json", body: emptyList }),
  );
  await page.route("**/api/v1/iocs/lookup**", (route) =>
    route.fulfill({ status: 404, body: JSON.stringify({ code: "not_found" }) }),
  );

  await page.goto("/iocs");
  await page.getByLabel("value").fill("1.2.3.4");
  await page.getByRole("button", { name: "Lookup" }).click();
  await expect(page.getByText("Not found")).toBeVisible();
});

test("ioc lookup renders a hit", async ({ page }) => {
  await page.route("**/api/v1/iocs?**", (route) =>
    route.fulfill({ status: 200, contentType: "application/json", body: emptyList }),
  );
  await page.route("**/api/v1/iocs/lookup**", (route) => {
    return route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        id: "abc",
        type: "domain",
        value: "evil.example",
        raw: "evil[.]example",
        firstSeenAt: "2026-01-01T00:00:00Z",
        lastSeenAt: "2026-05-01T00:00:00Z",
      }),
    });
  });

  await page.goto("/iocs");
  await page.getByLabel("value").fill("evil.example");
  await page.getByRole("button", { name: "Lookup" }).click();
  // The defang transform replaces dots; assert the rendered form.
  await expect(page.locator(".result-card dd code").first()).toHaveText("evil[.]example");
});

test("ioc recent list renders rows", async ({ page }) => {
  await page.route("**/api/v1/iocs?**", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        iocs: [
          {
            id: "ioc-1",
            type: "ipv4",
            value: "203.0.113.1",
            firstSeenAt: "2026-05-22T10:00:00Z",
            lastSeenAt: "2026-05-22T11:00:00Z",
          },
          {
            id: "ioc-2",
            type: "domain",
            value: "malicious.example",
            firstSeenAt: "2026-05-22T09:00:00Z",
            lastSeenAt: "2026-05-22T09:30:00Z",
          },
        ],
      }),
    }),
  );

  await page.goto("/iocs");
  await expect(page.getByRole("heading", { name: "Recent IoCs" })).toBeVisible();
  await expect(page.locator(".data-table tbody tr")).toHaveCount(2);
  await expect(page.locator(".data-table tbody tr").first()).toContainText("203[.]0[.]113[.]1");
});
