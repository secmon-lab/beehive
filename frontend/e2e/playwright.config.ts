import { defineConfig, devices } from "@playwright/test";

// The Playwright suite drives the React frontend with a real browser
// against a `beehive serve` instance running locally on port 8080.
//
// We expect the operator to bring beehive up themselves (e.g. via
// `go run . serve --repo-backend=memory`) — wiring the suite to manage
// the backend lifecycle is left for a follow-up so the config stays
// boring.

export default defineConfig({
  testDir: "./tests",
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL: process.env.BEEHIVE_E2E_BASE_URL ?? "http://localhost:5173",
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: process.env.BEEHIVE_E2E_BASE_URL
    ? undefined
    : {
        command: "pnpm dev",
        url: "http://localhost:5173",
        reuseExistingServer: !process.env.CI,
        stdout: "ignore",
        stderr: "pipe",
        timeout: 30_000,
      },
});
