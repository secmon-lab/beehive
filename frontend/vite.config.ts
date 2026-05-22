import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    // Pin the dev port so the Playwright webServer hook can wait on a
    // known URL and external testers know where the UI is.
    port: 5173,
    proxy: {
      "/api": {
        // `127.0.0.1` (not `localhost`) avoids the IPv6 resolution that
        // recent Node versions prefer — Go's net listener binds to both
        // stacks but local dev tooling sometimes only checks one.
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
        // Surface upstream connection errors instead of letting Vite
        // swallow them; makes "backend isn't running" obvious.
        configure: (proxy) => {
          proxy.on("error", (err, req) => {
            // eslint-disable-next-line no-console
            console.error("[vite proxy] %s %s -> %s", req.method, req.url, err.message);
          });
        },
      },
    },
  },
  build: {
    outDir: "dist",
    sourcemap: false,
    // Keep `dist/.gitkeep` (and any committed placeholder) intact across
    // `pnpm build` invocations so `//go:embed all:dist` still has a
    // non-empty directory in fresh checkouts that have not yet built
    // the frontend.
    emptyOutDir: false,
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test-setup.ts"],
    // Keep Vitest scoped to the unit tests under `src/`. The `e2e/`
    // tree is owned by Playwright (`pnpm test:e2e`).
    include: ["src/**/*.test.{ts,tsx}"],
  },
});
