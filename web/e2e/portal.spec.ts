import { test, expect } from '@playwright/test';

const admin = 'portal-admin@example.com';
const pass = 'password123';

test('authed portal groups tools and links to dashboard', async ({ page }) => {
  // Bootstrap admin (first account) and create a public tool.
  await page.goto('/signup');
  // On a fresh DB the app redirects /signup -> /setup once auth status loads;
  // wait for that to settle so the fields aren't cleared mid-fill.
  await page.waitForLoadState('networkidle');
  await page.fill('input[type=email]', admin);
  await page.fill('input[type=password]', pass);
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');

  // "Add for monitoring" opens the host picker; manual entry is the direct path.
  await page.goto('/services');
  await page.locator('button:text-is("Add for monitoring")').first().click();
  await page.locator('[role=dialog] button:text-is("Add manually")').click();
  await page.fill('input[required]', 'PortalGrafana');
  await page.fill('input[placeholder="10.0.0.5"]', '10.0.0.9');
  await page.click('button[type=submit]');
  await expect(page.locator('h1:has-text("PortalGrafana")')).toBeVisible();

  // Portal at root groups by collection; this tool is in none.
  await page.goto('/');
  await expect(page.locator('text=PortalGrafana')).toBeVisible();
  await expect(page.locator('h2:has-text("Ungrouped")')).toBeVisible();

  // Dashboard button enters the app with its top nav.
  await page.click('button:has-text("Dashboard")');
  await expect(page).toHaveURL('/dashboard');
  await expect(page.locator('nav a:has-text("Services")')).toBeVisible();

  // Clicking a tool opens its info modal; "Open service" navigates to detail.
  await page.goto('/');
  await page.click('text=PortalGrafana');
  await expect(page.getByRole('dialog', { name: 'Service' })).toBeVisible();
  await page.click('button:has-text("Open service")');
  await expect(page).toHaveURL(/\/services\/.+/);
});

test('anonymous visitor sees public tool and opens its info modal', async ({ page, context }) => {
  await context.clearCookies();
  await page.goto('/');
  await expect(page.locator('text=PortalGrafana')).toBeVisible();
  await page.click('text=PortalGrafana');
  // Anonymous gets a read-only info modal (no "Open service"), not a redirect.
  await expect(page.getByRole('dialog', { name: 'Service' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Open service' })).toHaveCount(0);
});

test('portal switches between list, map, and graph views', async ({ page, context }) => {
  await context.clearCookies();
  await page.goto('/');
  await page.click('button:has-text("Map")');
  await expect(page.locator('.leaflet-container')).toBeVisible();
  await page.click('button:has-text("Graph")');
  await expect(page.locator('main').getByText('PortalGrafana')).toBeVisible();
  await page.click('button:has-text("List")');
  await expect(page.locator('text=PortalGrafana')).toBeVisible();
});
