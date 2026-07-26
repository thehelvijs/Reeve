import { defineConfig } from '@playwright/test';

// Default the e2e backend to a port distinct from 8080, which is often held by
// a locally deployed Reeve instance; the vite dev server this config spawns
// inherits the same value from process.env, so both sides agree without the
// caller having to export anything.
process.env.REEVE_DEV_PORT ??= '8099';
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
  },
  webServer: [
    {
      command:
        `sh -c 'rm -f /tmp/reeve-e2e.db*; REEVE_MASTER_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA= REEVE_DB=/tmp/reeve-e2e.db REEVE_ADDR=127.0.0.1:${devPort} go run -ldflags "-X main.version=9.9.9" ./server'`,
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
