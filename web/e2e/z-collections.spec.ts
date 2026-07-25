import { test, expect, type Page } from '@playwright/test';

// Reuses the admin and the tools bootstrapped by core.spec.ts and portal.spec.ts
// (serial, shared DB).
const admin = 'admin@example.com';
const pass = 'password123';

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', admin);
  await page.fill('input[type=password]', pass);
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

async function createCollection(page: Page, name: string, visibility: 'public' | 'restricted', tool: string) {
  await page.goto('/collections');
  await page.click('button:has-text("New collection")');
  const dialog = page.getByRole('dialog', { name: 'New collection' });
  await dialog.locator('input[placeholder="Manufacturing"]').fill(name);
  await dialog.locator('select').selectOption(visibility);
  await dialog
    .locator('label')
    .filter({ has: page.getByText(tool, { exact: true }) })
    .locator('input[type=checkbox]')
    .check();
  await dialog.getByRole('button', { name: 'Save' }).click();
  await expect(page.locator(`h1:has-text("${name}")`)).toBeVisible();
}

test('create a collection and see the portal group by it', async ({ page }) => {
  await login(page);
  await createCollection(page, 'Manufacturing', 'public', 'Grafana');

  await expect(page.locator('text=1 service')).toBeVisible();

  await page.goto('/');
  await expect(page.locator('h2:has-text("Manufacturing")')).toBeVisible();
  await expect(page.locator('h2:has-text("Ungrouped")')).toBeVisible();
});

test('the collection heading opens its info modal', async ({ page }) => {
  await login(page);
  await page.goto('/');
  await page.click('h2:has-text("Manufacturing")');
  await expect(page.getByRole('dialog', { name: 'Collection' })).toBeVisible();
  await expect(page.getByRole('dialog', { name: 'Collection' }).getByText('Grafana', { exact: true })).toBeVisible();
});

test('the host toggle survives a reload', async ({ page }) => {
  await login(page);
  await page.goto('/');
  await page.click('button:has-text("Host")');
  await expect(page).toHaveURL(/by=host/);
  await expect(page.locator('h2:has-text("Unassigned")')).toBeVisible();

  await page.reload();
  await expect(page.locator('h2:has-text("Unassigned")')).toBeVisible();
  await expect(page.locator('h2:has-text("Manufacturing")')).toHaveCount(0);

  await page.click('button:has-text("Collection")');
  await expect(page).not.toHaveURL(/by=host/);
  await expect(page.locator('h2:has-text("Manufacturing")')).toBeVisible();
});

test('a restricted collection is hidden from anonymous, its services are not', async ({ page, context }) => {
  await login(page);
  await createCollection(page, 'Secret', 'restricted', 'PortalGrafana');

  await page.goto('/');
  await expect(page.locator('h2:has-text("Secret")')).toBeVisible();

  await context.clearCookies();
  await page.goto('/');
  await expect(page.locator('h2:has-text("Manufacturing")')).toBeVisible();
  await expect(page.locator('h2:has-text("Secret")')).toHaveCount(0);
  // The service itself stays public; it falls back to Ungrouped.
  await expect(page.locator('h2:has-text("Ungrouped")')).toBeVisible();
  await expect(page.locator('text=PortalGrafana')).toBeVisible();
});

test('the catalog filters by collection', async ({ page }) => {
  await login(page);
  await page.goto('/catalog');
  await expect(page.locator('text=PortalGrafana')).toBeVisible();
  await page.locator('select').first().selectOption({ label: 'Manufacturing' });
  await expect(page.locator('text=Grafana').first()).toBeVisible();
  await expect(page.locator('text=PortalGrafana')).toHaveCount(0);
});
