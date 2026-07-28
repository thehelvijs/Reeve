import { test, expect, type Page } from '@playwright/test';

// The invite path with no relay configured, which is what this instance is: the
// link has to come back in the response, or the admin has just created an account
// nobody can sign into. Then the group's moderator manages its members without
// being an admin anywhere.

async function login(page: Page, email: string, password: string) {
  await page.goto('/login');
  await page.fill('input[type=email]', email);
  await page.fill('input[type=password]', password);
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

test('an invited account sets its password from the link and signs in', async ({ page, browser }) => {
  await login(page, 'admin@example.com', 'password123');
  await page.goto('/admin/users');
  await page.click('button:has-text("Invite user")');
  const dialog = page.locator('[role="dialog"]');
  await dialog.locator('input[type=email]').fill('invited@example.com');
  await dialog.locator('input').nth(1).fill('Invited Person');
  await dialog.locator('button:has-text("Invite")').click();

  await expect(page.locator('text=No mail relay is configured')).toBeVisible();
  const link = await page.locator('p.font-mono').textContent();
  expect(link).toContain('/reset?token=');
  await page.click('button:has-text("Done")');
  await expect(page.locator('text=invited@example.com')).toBeVisible();

  // The link is absolute against the server's own origin; the suite drives the UI
  // through the Vite proxy, so only the path travels.
  const ctx = await browser.newContext();
  const invitee = await ctx.newPage();
  await invitee.goto(new URL(link!.trim()).pathname + new URL(link!.trim()).search);
  await invitee.fill('input[type=password]', 'invited-password');
  await invitee.click('button[type=submit]');
  await login(invitee, 'invited@example.com', 'invited-password');

  // A basic account moderates nothing yet, so the page it would use is not offered.
  await invitee.goto('/dashboard');
  await expect(invitee.locator('nav a:has-text("User groups")')).toHaveCount(0);

  // The admin puts them in a group as its moderator.
  await page.goto('/admin/groups');
  await page.click('button:has-text("New group")');
  await page.fill('input[placeholder="ops"]', 'e2e-moderated');
  await page.click('button:has-text("Create")');
  const card = page.locator('div.rounded-card', { hasText: 'e2e-moderated' }).first();
  await card.locator('input[type=email]').fill('invited@example.com');
  await card.locator('select').last().selectOption('moderator');
  await card.locator('button:has-text("Add")').click();
  await expect(card.locator('text=Invited Person')).toBeVisible();

  // Now the moderator has the page, sees that group only, and cannot stand down.
  await invitee.goto('/dashboard');
  await expect(invitee.locator('nav a:has-text("User groups")')).toBeVisible();
  await invitee.goto('/admin/groups');
  await expect(invitee.locator('h2:has-text("e2e-moderated")')).toBeVisible();
  await expect(invitee.locator('h2')).toHaveCount(1);
  await expect(invitee.locator('button:has-text("Remove")')).toBeDisabled();
  await expect(invitee.locator('button:has-text("New group")')).toHaveCount(0);
  await expect(invitee.locator('button:has-text("Delete")')).toHaveCount(0);

  await ctx.close();

  // Both are removed again: an extra account and an extra group would show up in
  // every later spec's lists and in the visual baselines.
  const groups = await (await page.request.get('/api/groups')).json();
  const group = groups.find((g: { name: string }) => g.name === 'e2e-moderated');
  expect(await (await page.request.delete(`/api/admin/groups/${group.id}`)).ok()).toBe(true);
  const users = await (await page.request.get('/api/admin/users')).json();
  const invited = users.find((u: { email: string }) => u.email === 'invited@example.com');
  expect(await (await page.request.delete(`/api/admin/users/${invited.id}`)).ok()).toBe(true);
});
