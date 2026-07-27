import { test, expect, type Page } from '@playwright/test';

// The theme is one attribute on <html>, set before first paint by
// public/theme-boot.js and changed by the toggle. What matters is that the
// preference survives a reload (otherwise the toggle is a per-page trick) and
// that an anonymous visitor can reach it at all, since the portal is the front
// door and has no profile page.

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

function theme(page: Page) {
  return page.locator('html');
}

test('dark is the default with nothing stored', async ({ page, context }) => {
  await context.clearCookies();
  await page.goto('/login');
  await expect(theme(page)).toHaveAttribute('data-theme', 'dark');
});

test('the toggle switches the app and the choice survives a reload', async ({ page }) => {
  await login(page);
  await page.goto('/dashboard');
  await expect(theme(page)).toHaveAttribute('data-theme', 'dark');

  await page.getByRole('button', { name: 'Switch to light theme' }).click();
  await expect(theme(page)).toHaveAttribute('data-theme', 'light');

  // The boot script, not React, has to apply this on the next load.
  await page.reload();
  await expect(theme(page)).toHaveAttribute('data-theme', 'light');
  await expect(page.getByRole('button', { name: 'Switch to dark theme' })).toBeVisible();

  await page.getByRole('button', { name: 'Switch to dark theme' }).click();
  await expect(theme(page)).toHaveAttribute('data-theme', 'dark');
});

test('an anonymous visitor can switch from the portal', async ({ page, context }) => {
  await context.clearCookies();
  await page.route(/basemaps\.cartocdn\.com/, (route) => route.abort());
  await page.goto('/');
  await expect(page.locator('main')).toBeVisible();

  await page.getByRole('button', { name: 'Switch to light theme' }).click();
  await expect(theme(page)).toHaveAttribute('data-theme', 'light');
  await page.reload();
  await expect(theme(page)).toHaveAttribute('data-theme', 'light');
});

// The meta tag colors the browser's own chrome on mobile, so a stale value shows
// a white bar over a dark page.
test('theme-color follows the theme', async ({ page }) => {
  await login(page);
  await page.goto('/dashboard');
  const meta = page.locator('meta[name="theme-color"]');
  await expect(meta).toHaveAttribute('content', '#1f1e24');
  await page.getByRole('button', { name: 'Switch to light theme' }).click();
  await expect(meta).toHaveAttribute('content', '#ffffff');
});
