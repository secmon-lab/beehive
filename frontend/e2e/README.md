# Frontend E2E (Playwright)

The browser-side end-to-end suite. Each spec drives the React UI with a
real Chromium and asserts on the rendered DOM.

## Layout

```
e2e/
├── playwright.config.ts      # config (uses pnpm dev as the webServer)
├── tests/
│   ├── sources.spec.ts       # /sources mounts, nav links visible, / -> /sources
│   ├── iocs.spec.ts          # lookup form, hit & miss flows (API is stubbed)
│   └── runs.spec.ts          # /runs renders the empty state
```

API calls are intercepted with `page.route(...)`, so the suite does not
need a running `beehive serve`. To point it at a real backend instead,
set `BEEHIVE_E2E_BASE_URL` — Playwright will skip the bundled `pnpm dev`
server and the route handlers must be removed from the affected specs.

## Running

```bash
pnpm install
pnpm exec playwright install chromium
pnpm test:e2e
```

The CI workflow runs the same command. Specs are kept small on purpose:
the goal is to catch "page does not mount" regressions, not to cover
every interaction (that lives in the Vitest unit tests).
