import { test, expect } from '@playwright/test';

// Enter has to confirm every editable section, not just the ones that happen to
// be a <form> already. Runs late (zz-) so admin@example.com and PortalGrafana
// from the earlier specs exist.
async function loginAdmin(page: import('@playwright/test').Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

test('enter saves the display name without clicking Save', async ({ page }) => {
  await page.goto('/signup');
  await page.waitForLoadState('networkidle');
  await page.fill('input[type=email]', 'enter-key@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');

  await page.goto('/profile');
  const name = page.getByRole('textbox', { name: /Shown across the app/ });
  await name.fill('Enter Person');
  await name.press('Enter');
  await expect(page.getByText('Saved.')).toBeVisible();
  await expect(page.locator('aside')).toContainText('Enter Person');

  await page.getByRole('textbox', { name: 'Current password' }).fill('password123');
  await page.getByRole('textbox', { name: 'New password', exact: true }).fill('newpassword1');
  const confirmPw = page.getByRole('textbox', { name: 'Confirm new password' });
  await confirmPw.fill('newpassword1');
  await confirmPw.press('Enter');
  await expect(page.getByText('Password updated.')).toBeVisible();
});

test('enter saves alert thresholds and adds a webhook', async ({ page }) => {
  await loginAdmin(page);

  await page.goto('/admin/alerts');
  const window = page.getByRole('spinbutton', { name: 'Window (minutes)' });
  await window.fill('7');
  await window.press('Enter');
  await expect(page.getByText('Saved.')).toBeVisible();

  await page.goto('/admin/webhooks');
  const url = page.getByRole('textbox', { name: 'URL' });
  await url.fill('https://enter.example.com/hook');
  await url.press('Enter');
  await expect(page.getByText('https://enter.example.com/hook')).toBeVisible();
});

test('enter confirms a portal info modal', async ({ page }) => {
  await loginAdmin(page);
  await page.goto('/');
  await page.click('text=PortalGrafana');
  await expect(page.getByRole('dialog', { name: 'Service' })).toBeVisible();
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(/\/catalog\/.+/);
});
