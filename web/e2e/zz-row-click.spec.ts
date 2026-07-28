import { test, expect, type Page } from '@playwright/test';

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

// The host is enrolled for these two clicks and deleted again: left behind it
// would be one more row in every later spec's fleet, and the visual baselines
// describe a settled one.
test('a row click opens the record, a row action does not', async ({ page }) => {
  await login(page);
  const created = await page.request.post('/api/admin/hosts', { data: { name: 'row-click-host' } });
  expect(created.status()).toBe(201);
  const host = await created.json();
  const row = page.locator('tbody tr', { hasText: 'row-click-host' });

  await page.goto('/hosts');
  await expect(row).toBeVisible();
  // The OS cell holds no control, so the click belongs to the row.
  await row.locator('td').nth(1).click();
  await expect(page).toHaveURL(`/hosts/${host.host.id}`);

  // Without the guard in Tr the row would navigate out from under the menu this
  // click just opened.
  await page.goto('/hosts');
  await expect(row).toBeVisible();
  await row.locator('button').first().click();
  await expect(page).toHaveURL('/hosts');

  const gone = await page.request.delete(`/api/admin/hosts/${host.host.id}`);
  expect(gone.ok(), 'the row-click host outlived its test').toBe(true);
});
