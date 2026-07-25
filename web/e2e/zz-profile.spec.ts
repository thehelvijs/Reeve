import { test, expect } from '@playwright/test';

// Runs last (zz- prefix) so the bootstrap admin from core.spec already exists
// and signups here create ordinary secondary accounts.
async function signup(page: import('@playwright/test').Page, email: string, pass: string) {
  await page.goto('/signup');
  await page.waitForLoadState('networkidle');
  await page.fill('input[type=email]', email);
  await page.fill('input[type=password]', pass);
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

test('profile: change display name and password', async ({ page }) => {
  await signup(page, 'profile-user@example.com', 'password123');

  await page.goto('/profile');
  await page.getByRole('textbox', { name: /Shown across the app/ }).fill('Profile Person');
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByText('Saved.')).toBeVisible();
  await expect(page.locator('aside')).toContainText('Profile Person');

  await page.getByRole('textbox', { name: 'Current password' }).fill('password123');
  await page.getByRole('textbox', { name: 'New password', exact: true }).fill('newpassword1');
  await page.getByRole('textbox', { name: 'Confirm new password' }).fill('newpassword1');
  await page.getByRole('button', { name: 'Update password' }).click();
  await expect(page.getByText('Password updated.')).toBeVisible();

  await page.click('button:has-text("Sign out")');
  await page.goto('/login');
  await page.fill('input[type=email]', 'profile-user@example.com');
  await page.fill('input[type=password]', 'newpassword1');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
  await page.goto('/dashboard');
  await expect(page.locator('nav a:has-text("Services")')).toBeVisible();
});

test('profile: self-delete account', async ({ page }) => {
  await signup(page, 'delete-me@example.com', 'password123');

  await page.goto('/profile');
  await page.getByRole('textbox', { name: /Type .* to confirm/ }).fill('delete-me@example.com');
  await page.getByRole('button', { name: 'Delete account' }).click();
  await expect(page).toHaveURL('/');

  await page.goto('/login');
  await page.fill('input[type=email]', 'delete-me@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page.getByText('invalid email or password')).toBeVisible();
});
