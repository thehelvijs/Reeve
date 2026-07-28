import { test, expect } from '@playwright/test';

// Enter has to confirm every editable section, not just the ones that happen to
// be a <form> already. Runs late (zz-) so admin@example.com and PortalGrafana
// from the earlier specs exist.
async function loginAdmin(page: import('@playwright/test').Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  // Enter in the last field of the sign-in form is the login itself.
  await page.locator('input[type=password]').fill('password123');
  await page.locator('input[type=password]').press('Enter');
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
  await expect(page).toHaveURL(/\/services\/.+/);
});

test('enter saves settings sections', async ({ page }) => {
  await loginAdmin(page);
  await page.goto('/admin/settings');
  // Every section re-syncs its draft from the settings response, so typing before
  // the page has finished loading is typing into a field about to be overwritten.
  await page.waitForLoadState('networkidle');

  // Each section saves through the same PUT, so wait for it rather than racing
  // the reload against an in-flight request.
  const saved = () =>
    page.waitForResponse((r) => r.url().endsWith('/api/admin/settings') && r.request().method() === 'PUT');

  const concurrency = page.getByRole('spinbutton', { name: 'Hosts updating at once' });
  await concurrency.fill('7');
  await Promise.all([saved(), concurrency.press('Enter')]);
  await page.reload();
  await expect(page.getByRole('spinbutton', { name: 'Hosts updating at once' })).toHaveValue('7');

  const smtpHost = page.getByRole('textbox', { name: 'Host' });
  await smtpHost.fill('smtp.enter.example.com');
  await Promise.all([saved(), smtpHost.press('Enter')]);
  await page.reload();
  await expect(page.getByRole('textbox', { name: 'Host' })).toHaveValue('smtp.enter.example.com');

  const domains = page.getByRole('textbox', { name: 'Allowed email domains' });
  await domains.fill('enter.example.com');
  await Promise.all([saved(), domains.press('Enter')]);
  await page.reload();
  await expect(page.getByRole('textbox', { name: 'Allowed email domains' })).toHaveValue('enter.example.com');
});

test('enter creates a host from the add-host modal', async ({ page }) => {
  await loginAdmin(page);
  await page.goto('/hosts');
  await page.getByRole('button', { name: 'Add host' }).first().click();
  // The modal focuses its first field on open, so Enter needs no click first.
  await page.keyboard.type('enter-key-host');
  await page.keyboard.press('Enter');
  await expect(page.getByText('Host created.')).toBeVisible();
});

test('enter saves the service form, and the nested collection field keeps it open', async ({ page }) => {
  await loginAdmin(page);
  await page.goto('/services/new');

  const newCollection = page.getByPlaceholder('New collection');
  await newCollection.fill('EnterColl');
  await newCollection.press('Enter');
  await expect(page.getByText('EnterColl')).toBeVisible();
  await expect(page).toHaveURL('/services/new');

  const name = page.locator('input[required]');
  await name.fill('EnterService');
  await page.getByPlaceholder('10.0.0.5').fill('10.0.0.42');
  await name.press('Enter');
  await expect(page.locator('h1:has-text("EnterService")')).toBeVisible();
});

test('enter stores and reveals a credential', async ({ page }) => {
  await loginAdmin(page);
  await page.goto('/hosts');
  await page.getByRole('button', { name: 'Add host' }).click();
  await page.getByPlaceholder('db-server-1').fill('EnterHost');
  await page.getByRole('button', { name: 'Create' }).click();
  await page.getByRole('button', { name: 'Done' }).click();
  await page.click('text=EnterHost');
  await page.getByRole('tab', { name: 'Credentials' }).click();
  await page.getByRole('button', { name: 'Add credential' }).click();
  await page.getByRole('textbox', { name: 'Label' }).fill('enter-cred');
  await page.getByRole('textbox', { name: 'username' }).fill('root');
  const pw = page.getByRole('dialog').locator('input[type=password]');
  await pw.fill('hunter2');
  await pw.press('Enter');
  await expect(page.getByText('enter-cred')).toBeVisible();

  await page.getByRole('button', { name: 'Reveal' }).click();
  const reveal = page.getByRole('dialog', { name: 'enter-cred' });
  await expect(reveal).toBeVisible();
  await page.keyboard.press('Enter');
  await expect(reveal).toHaveCount(0);
});

test('enter creates a collection from its editor', async ({ page }) => {
  await loginAdmin(page);
  await page.goto('/collections');
  await page.getByRole('button', { name: 'New collection' }).first().click();
  await page.keyboard.type('EnterEditorColl');
  await page.keyboard.press('Enter');
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.getByText('EnterEditorColl')).toBeVisible();
});

test('enter resets a password from the admin modal', async ({ page }) => {
  await loginAdmin(page);
  await page.goto('/admin/users');
  const row = page.locator('div.rounded-card', { hasText: 'enter-key@example.com' });
  await row.getByRole('button', { name: 'Reset password' }).click();
  await page.keyboard.type('resetbyenter1');
  await page.keyboard.press('Enter');
  await expect(page.getByRole('dialog', { name: 'Password reset' })).toBeVisible();
});

test('enter advances the how-to modal and probes over SSH', async ({ page }) => {
  await loginAdmin(page);
  await page.goto('/hosts');

  await page.getByRole('button', { name: 'How to add a host' }).click();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('dialog', { name: 'Add host' })).toBeVisible();
  await page.keyboard.press('Escape');

  await page
    .locator('#host-list tr', { hasText: 'enter-key-host' })
    .getByRole('button', { name: 'Install over SSH' })
    .click();
  // Address is focused on open; a closed port fails the probe fast.
  await page.keyboard.type('127.0.0.1');
  await page.getByRole('spinbutton', { name: 'Port' }).fill('1');
  await page.getByRole('textbox', { name: 'Username' }).fill('nobody');
  const [probe] = await Promise.all([
    page.waitForResponse((r) => r.url().endsWith('/api/admin/ssh-probe')),
    page.getByRole('textbox', { name: 'Username' }).press('Enter'),
  ]);
  expect(probe.status()).toBe(502);
  await expect(page.getByRole('dialog').locator('p.text-down')).toBeVisible();
});

test('enter picks a city and then saves the host location', async ({ page }) => {
  await loginAdmin(page);
  await page.goto('/hosts');
  await page.locator('#host-list a', { hasText: 'enter-key-host' }).first().click();
  await page.getByRole('tab', { name: 'Settings' }).click();

  // The address geocoder is stubbed out: what is under test is Enter, and the
  // bundled city list answers "Riga" without leaving the machine.
  await page.route('**/api/admin/geocode**', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: '[]' }),
  );
  const city = page.getByPlaceholder(/^Search a city or address/);
  await city.fill('Riga');
  await city.press('Enter');
  await expect(city).toHaveValue(/^Riga, /);
  await city.press('Enter');
  await expect(page.getByText('Saved.')).toBeVisible();
});
