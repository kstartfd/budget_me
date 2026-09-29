import { defineConfig, devices } from "@playwright/test"

/**
 * Playwright config for the budget_me frontend.
 *
 * The test stack:
 *  - test backend at http://localhost:8181 (started via `make test.up` —
 *    isolated Postgres),
 *  - test frontend at http://localhost:3001 (Playwright auto-starts
 *    `pnpm dev` on port 3001 with NEXT_PUBLIC_API_URL pointing at 8181).
 *
 * Only Chromium is configured. Firefox/WebKit can be added later once the
 * suite stabilises.
 */
const FRONTEND_PORT = process.env.E2E_FRONTEND_PORT ?? "3001"
const BACKEND_URL = process.env.E2E_BACKEND_URL ?? "http://localhost:8181"
const BASE_URL = `http://localhost:${FRONTEND_PORT}`

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: process.env.CI ? "github" : "list",

  use: {
    baseURL: BASE_URL,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
    locale: "en-US",
    launchOptions: {
      executablePath: process.env.E2E_CHROMIUM_PATH,
    },
    extraHTTPHeaders: {
      "Accept-Language": "en-US,en;q=0.9",
    },
  },

  projects: [
    // Anon-context flows (signup, login) — fresh browser per test.
    {
      name: "chromium-anon",
      testMatch: /.*\.anon\.spec\.ts/,
      use: { ...devices["Desktop Chrome"] },
    },
    // Auth-context flows — each test calls signUpAsNewUser() in its own
    // beforeEach, so we don't need a shared storageState fixture. That
    // gives full isolation between tests (no state leaks through the
    // server-side profile).
    {
      name: "chromium-auth",
      testMatch: /.*\.auth\.spec\.ts/,
      use: { ...devices["Desktop Chrome"] },
    },
  ],

  webServer: {
    // Frontend reads BACKEND_URL for its API proxy, so we set it here to
    // route every API call at the isolated test backend on :8181. PORT
    // keeps the dev server out of the way of any human-driven `pnpm dev`
    // running on :3000.
    //
    // reuseExistingServer is true: Playwright's "couldn't kill the dev
    // server from the previous run" leaves the :3001 port held by a
    // stale process, which would otherwise wedge every subsequent
    // invocation. Turbopack hot-reloads when source changes, so reusing
    // an already-running dev server is safe between code edits too.
    command: `PORT=${FRONTEND_PORT} BACKEND_URL=${BACKEND_URL} pnpm dev`,
    url: BASE_URL,
    reuseExistingServer: true,
    timeout: 180 * 1000,
    stdout: "ignore",
    stderr: "pipe",
  },
})
