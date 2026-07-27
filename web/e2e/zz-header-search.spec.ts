import { test, expect, type Page } from '@playwright/test';

// Every list page carries a search in its own header that filters the rows in
// place. What has to hold: the filter narrows the list rather than reloading it,
// two terms narrow further instead of widening, and Cmd/Ctrl+K still means the
// sidebar's global search on a page that now has a second box.
//
// Runs after the specs that enroll hosts (core, z-), so the table has rows.

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

async function hostRows(page: Page) {
  await login(page);
  await page.goto('/hosts');
  const rows = page.locator('#host-list tbody tr');
  await expect(rows.first()).toBeVisible();
  return rows;
}

function hostSearch(page: Page) {
  return page.getByRole('searchbox', { name: 'Search hosts…' });
}

test('the hosts header search narrows the table and restores it', async ({ page }) => {
  const rows = await hostRows(page);
  const all = await rows.count();
  const name = (await rows.first().locator('a').first().innerText()).trim();

  await hostSearch(page).fill(name);
  await expect(rows.filter({ hasNotText: name })).toHaveCount(0);
  expect(await rows.count()).toBeGreaterThan(0);

  await hostSearch(page).fill('');
  await expect(rows).toHaveCount(all);
});

// Every term has to hit, or typing a second word would grow the list.
test('two terms narrow instead of widening', async ({ page }) => {
  const rows = await hostRows(page);
  const name = (await rows.first().locator('a').first().innerText()).trim();

  await hostSearch(page).fill(`${name} linux`);
  expect(await rows.count()).toBeGreaterThan(0);

  await hostSearch(page).fill(`${name} plan9`);
  await expect(rows).toHaveCount(0);
  await expect(page.getByText('No host in this state matches the search.')).toBeVisible();
});

// The page search deliberately has no shortcut: one Cmd+K, one meaning. The
// event is dispatched rather than typed because Chrome keeps Ctrl+K for itself.
test('Cmd+K still focuses the global search', async ({ page }) => {
  await hostRows(page);
  await page.evaluate(() =>
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true, bubbles: true })),
  );
  await expect(page.getByRole('searchbox', { name: 'Search services…' })).toBeFocused();
});

test('a search with no match says so instead of showing an empty page', async ({ page }) => {
  await login(page);
  await page.goto('/admin/users');
  await page.getByRole('searchbox', { name: 'Search users…' }).fill('nobody@nowhere.invalid');
  await expect(page.getByText('No user matches the search.')).toBeVisible();
});
