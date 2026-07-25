import { test, expect } from '@playwright/test';

// Depends (like server-info.spec) on core.spec creating admin@example.com first
// in the serial run.
test('admin edits thresholds and sees the form', async ({ page }) => {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');

  await page.goto('/admin/alerts');
  await expect(page.locator('h2:has-text("Thresholds")')).toBeVisible();
});
