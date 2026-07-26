import { defineConfig } from '@playwright/test';

// The suite always owns this port outright (not ??=) so it can't collide with
// a REEVE_DEV_PORT a developer already exports for their own dev server; the
// vite dev server this config spawns inherits the overridden value.
process.env.REEVE_DEV_PORT = '8099';
const devPort = process.env.REEVE_DEV_PORT;

// Pre-ship e2e. Starts a fresh Go server (throwaway DB, fixed test master key)
// and the Vite dev server, then drives the core flows through the proxy.
export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  fullyParallel: false,
  workers: 1,
  use: {
    baseURL: 'http://127.0.0.1:5173',
    trace: 'on-first-retry',
    channel: 'chrome',
    // Specs that drive the API directly reuse the browser's session cookie, so
    // they have to look like the browser to the server's same-origin check. A
    // page's own requests already carry this same value.
    extraHTTPHeaders: { Origin: 'http://127.0.0.1:5173' },
  },
  webServer: [
    {
      command:
        // The UI is served by Vite on another port and proxied here, so its
        // origin has to be allowed past the server's same-origin check.
        `sh -c 'rm -f /tmp/reeve-e2e.db*; REEVE_MASTER_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA= REEVE_DB=/tmp/reeve-e2e.db REEVE_ADDR=127.0.0.1:${devPort} REEVE_ALLOWED_ORIGINS=http://127.0.0.1:5173,http://localhost:5173 go run -ldflags "-X main.version=9.9.9" ./server'`,
      cwd: '..',
      url: `http://127.0.0.1:${devPort}/healthz`,
      reuseExistingServer: false,
      timeout: 60_000,
    },
    {
      command: 'npm run dev',
      url: 'http://127.0.0.1:5173',
      reuseExistingServer: false,
      timeout: 60_000,
    },
  ],
});
