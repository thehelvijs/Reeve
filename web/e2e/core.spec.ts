import { test, expect } from '@playwright/test';

// A unique email per run so the suite is re-runnable against a fresh DB.
const admin = `admin@example.com`;
const pass = 'password123';

test('signup, create tool, store and reveal a host credential', async ({ page }) => {
  // First account bootstraps as admin.
  await page.goto('/signup');
  // On a fresh DB the app redirects /signup -> /setup once auth status loads;
  // wait for that to settle so the fields aren't cleared mid-fill.
  await page.waitForLoadState('networkidle');
  await page.fill('input[type=email]', admin);
  await page.fill('input[type=password]', pass);
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');

  // Create a tool via the dedicated form page.
  await page.goto('/catalog');
  await page.click('button:has-text("Add for monitoring")');
  await expect(page).toHaveURL('/catalog/new');
  await page.fill('input[required]', 'Grafana');
  await page.fill('input[placeholder="10.0.0.5"]', '10.0.0.9');
  await page.click('button[type=submit]');
  // Saving lands on the new tool's detail page.
  await expect(page.locator('h1:has-text("Grafana")')).toBeVisible();

  // Credentials hang off the machine, not the service, so they are added on
  // the host page.
  await page.goto('/hosts');
  await page.click('button:has-text("Add host")');
  await page.fill('input[placeholder="db-server-1"]', 'cred-host');
  await page.click('button:has-text("Create")');
  // Creating shows the enrollment token modal; dismiss it before navigating.
  await page.click('button:has-text("Done")');
  await page.click('text=cred-host');

  await page.click('button:has-text("Add credential")');
  // Default type is ssh_password; switch to api_token for a single field.
  await page.selectOption('select', 'api_token');
  // The token field is the only explicit type=text input in the modal.
  await page.fill('input[type=text]', 'secret-token-xyz');
  await page.click('button:has-text("Save credential")');

  // Reveal it and assert the secret is shown.
  await page.click('button:has-text("Reveal")');
  await expect(page.locator('pre:has-text("secret-token-xyz")')).toBeVisible();
});

test('enroll a host and see it listed', async ({ page }) => {
  await page.goto('/login');
  await page.fill('input[type=email]', admin);
  await page.fill('input[type=password]', pass);
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');

  await page.goto('/hosts');
  await page.click('button:has-text("Add host")');
  await page.fill('input[placeholder="db-server-1"]', 'e2e-host');
  await page.click('button:has-text("Create")');
  // The run command with the enrollment token is shown once.
  await expect(page.locator('pre:has-text("REEVE_AGENT_TOKEN")').first()).toBeVisible();
  await page.click('button:has-text("Done")');
  await expect(page.locator('text=e2e-host')).toBeVisible();
});
