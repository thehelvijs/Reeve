import { test, expect } from '@playwright/test';

// The suite runs serially against a shared DB; core.spec creates the first
// account (admin@example.com), so log in as that admin rather than signing up a
// later (basic-role) user.
// The server samples itself every 30s, and a chart needs two points, so on a
// freshly started instance the grid legitimately says so for the first half
// minute. This waits for the second sample rather than asserting whichever
// state the suite happened to arrive in.
test('admin sees the server info page', async ({ page }) => {
  test.setTimeout(90_000);
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');

  await page.goto('/admin/server');
  await expect(page.locator('h1:has-text("Server")')).toBeVisible();
  await expect(page.locator('text=Version')).toBeVisible();
  // Self-monitoring: live gauges + the metric chart grid.
  await expect(page.locator('h2:has-text("Charts")')).toBeVisible();
  await expect(page.getByText('Load average')).toBeVisible({ timeout: 50_000 });
});
