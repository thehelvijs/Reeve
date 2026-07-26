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
  // KPI strip.
  await expect(page.locator('text=Hosts online')).toBeVisible();
  await expect(page.locator('text=To review')).toBeVisible();
  // Host card for the host core.spec enrolled (no samples -> placeholder).
  await expect(page.locator('text=e2e-host')).toBeVisible();
  await expect(page.locator('text=No metrics yet').first()).toBeVisible();
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
