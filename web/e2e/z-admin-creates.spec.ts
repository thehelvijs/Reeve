import { test, expect } from '@playwright/test';

// Reuses the admin bootstrapped by core.spec.ts (serial, shared DB).
const admin = 'admin@example.com';
const pass = 'password123';

async function login(page: import('@playwright/test').Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', admin);
  await page.fill('input[type=password]', pass);
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

test('dashboard shows KPI strip and a per-host card', async ({ page }) => {
  await login(page);
  await page.goto('/dashboard');
  // Every card on this page earns its place: core.spec's host has never pushed,
  // so the hosts card is there, and no access request has been raised yet, so the
  // review card is not.
  await expect(page.locator('text=Hosts online')).toBeVisible();
  await expect(page.locator('text=To review')).toHaveCount(0);
  // Host card for the host core.spec enrolled (no samples -> placeholder).
  await expect(page.locator('main').getByText('e2e-host')).toBeVisible();
  await expect(page.locator('text=No metrics yet').first()).toBeVisible();
});

// Both enroll commands are runnable as shown, with no REEVE_PUBLIC_URL set on
// this instance. Which host header the server derives the URL from is covered by
// TestAgentCommandsFallBackToRequestHost — it cannot be asserted from here,
// because Vite proxies /api to the Go server and replaces the Host header with
// its own target on the way through. What this does catch is a command that
// names an address no agent could dial, which is what the old cfg.Addr fallback
// produced.
test('both enroll commands are runnable as shown', async ({ page }) => {
  await login(page);
  await page.goto('/hosts');
  await page.click('button:has-text("Add host")');
  await page.fill('input[placeholder="db-server-1"]', 'e2e-enroll-url');
  await page.click('button:has-text("Create")');

  const install = page.locator('pre', { hasText: 'install.sh' });
  const docker = page.locator('pre', { hasText: 'docker run' });
  await expect(install).toBeVisible();
  await expect(docker).toBeVisible();

  const origin = 'http://127.0.0.1:8099';
  await expect(install).toContainText(`curl -fsSL ${origin}/install.sh`);
  for (const block of [install, docker]) {
    await expect(block).toContainText(`REEVE_SERVER_URL=${origin}`);
    await expect(block).toContainText(/REEVE_AGENT_TOKEN=rva_[0-9a-f]{64}/);
    // An unspecified or empty host is the failure the old fallback produced: it
    // renders as a URL and is useless to every machine but this one.
    await expect(block).not.toContainText('0.0.0.0');
    await expect(block).not.toContainText('://:');
  }
  await page.click('button:has-text("Done")');
});

test('create a group via the header modal', async ({ page }) => {
  await login(page);
  await page.goto('/admin/groups');
  await page.click('button:has-text("New group")');
  await page.fill('input[placeholder="ops"]', 'e2e-group');
  await page.click('button:has-text("Create")');
  await expect(page.locator('h2:has-text("e2e-group")')).toBeVisible();
});

test('enter submits a modal', async ({ page }) => {
  await login(page);
  await page.goto('/admin/groups');
  await page.click('button:has-text("New group")');
  await page.fill('input[placeholder="ops"]', 'e2e-enter-group');
  await page.keyboard.press('Enter');
  await expect(page.locator('h2:has-text("e2e-enter-group")')).toBeVisible();
});

test('create a webhook', async ({ page }) => {
  await login(page);
  await page.goto('/admin/webhooks');
  await page.fill('input[placeholder="https://sink.example.com/hook"]', 'https://example.com/hook');
  await page.click('button:has-text("Add webhook")');
  await expect(page.locator('text=https://example.com/hook')).toBeVisible();
});
