import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

// Three behaviours that were only ever checked by hand: the confirm in front of a
// destructive row action, the per-service visibility toggle, and the filesystem
// list. Each is something a person asked for and each was verified in a browser
// once, which is not the same as being verified on every commit.
//
// Runs after zzz-visual (zzzz- sorts last) because it enrolls hosts, and a new
// row would move the hosts baseline.

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

// A host that reports control support, inventory and filesystems, and stays
// online long enough for the assertions (offline_after_secs well past the run).
async function enrollReportingHost(req: APIRequestContext, name: string) {
  const created = await req.post('/api/admin/hosts', { data: { name, offline_after_secs: 3600 } });
  expect(created.status()).toBe(201);
  const body = await created.json();
  const push = await req.post('/api/ingest', {
    headers: { Authorization: `Bearer ${body.enroll_token}` },
    data: {
      agent_version: '9.9.9',
      agent_checksum: '0'.repeat(64),
      control_enabled: true,
      sent_at: new Date().toISOString(),
      services: [{ unit: 'nginx.service', active_state: 'active', sub_state: 'running' }],
      containers: [],
      metrics: {
        cpu_pct: 5,
        mem_used: 500,
        mem_total: 1000,
        disk_used: 16,
        disk_total: 30,
        disks: [
          { mount: '/', device: '/dev/sda2', fs_type: 'ext4', used: 16, total: 30 },
          { mount: '/mnt/media', device: '/dev/sdb1', fs_type: 'ext4', used: 95, total: 100 },
        ],
      },
    },
  });
  expect(push.status()).toBe(200);
  return body.host.id as string;
}

test('a destructive row action asks first, and cancelling runs nothing', async ({ page }) => {
  await login(page);
  const hostID = await enrollReportingHost(page.request, 'e2e-confirm-host');
  await page.goto(`/hosts/${hostID}?tab=inventory`);

  // Restart takes whatever is using the unit down with it, so it asks.
  await page.locator('button:text-is("Restart")').first().click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  // The question names the target: these rows sit close together.
  await expect(dialog).toContainText('nginx.service');
  await expect(dialog).toContainText('e2e-confirm-host');

  await dialog.getByRole('button', { name: 'Cancel' }).click();
  await expect(dialog).toHaveCount(0);
  const afterCancel = await page.request.get(`/api/admin/hosts/${hostID}/commands`);
  expect(await afterCancel.json()).toEqual([]);

  // Confirming queues it, which is the other half of the guarantee.
  await page.locator('button:text-is("Restart")').first().click();
  await page.getByRole('dialog').getByRole('button', { name: 'Restart' }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect
    .poll(async () => (await (await page.request.get(`/api/admin/hosts/${hostID}/commands`)).json()).length)
    .toBe(1);

  // Starting something that is down cannot lose anything, so it stays one click.
  await page.locator('button:text-is("Start")').first().click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
});

test('the visibility toggle hides a service from the anonymous portal', async ({ page, context }) => {
  await login(page);
  const created = await page.request.post('/api/tools', {
    data: {
      name: 'E2EToggle',
      scheme: 'http',
      address: '10.0.0.77',
      port: 8080,
      source_type: 'manual',
      visibility: 'public',
    },
  });
  expect(created.status()).toBe(201);

  await page.goto('/services');
  // The visibility control is its own cell beside the name link, so the row —
  // not the link — is what scopes it.
  const row = page.locator('tr', { hasText: 'E2EToggle' });
  await expect(row.getByRole('button', { name: 'public' })).toBeVisible();

  // Anonymous sees it while it is public.
  const anon = await context.browser()!.newContext();
  const anonPage = await anon.newPage();
  await anonPage.goto('/');
  await expect(anonPage.locator('text=E2EToggle')).toBeVisible();

  // One click, no edit form.
  await row.getByRole('button', { name: 'public' }).click();
  await expect(row.getByRole('button', { name: 'restricted' })).toBeVisible();

  await anonPage.reload();
  await expect(anonPage.locator('text=E2EToggle')).toHaveCount(0);

  // And back, so the toggle is a toggle rather than a one-way door.
  await row.getByRole('button', { name: 'restricted' }).click();
  await expect(row.getByRole('button', { name: 'public' })).toBeVisible();
  await anonPage.reload();
  await expect(anonPage.locator('text=E2EToggle')).toBeVisible();
  await anon.close();
});

test('every reported filesystem is listed, fullest first', async ({ page }) => {
  await login(page);
  const hostID = await enrollReportingHost(page.request, 'e2e-disk-host');
  await page.goto(`/hosts/${hostID}?tab=metrics`);

  const card = page.locator('section').filter({ has: page.locator('h2:text-is("Filesystems")') });
  await expect(card).toContainText('Filesystems (2)');
  await expect(card).toContainText('/mnt/media · /dev/sdb1 · ext4');
  await expect(card).toContainText('/ · /dev/sda2 · ext4');

  // 95% full comes before 53%: the reason to read the list is finding the one
  // about to run out, and that is rarely the root filesystem.
  const labels = await card.locator('span.text-muted').allInnerTexts();
  const media = labels.findIndex((l) => l.includes('/mnt/media'));
  const root = labels.findIndex((l) => l.startsWith('/ ·'));
  expect(media, 'both filesystems are labelled').toBeGreaterThanOrEqual(0);
  expect(media).toBeLessThan(root);

  // The server's own filesystems are the same feature on a different page.
  await page.goto('/admin/server');
  await expect(page.getByRole('heading', { name: /^Filesystems/ })).toBeVisible();
});
