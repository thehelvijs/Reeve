import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

// A busy machine reports hundreds of units, containers and cron jobs into three
// scrolling lists, plus a process table. Each list carries its own search in its
// heading. What has to hold: the search filters that list and nothing else, the
// heading count follows it, and an empty result says why.
//
// Runs after zzz-visual (zzzz- sorts last) because it enrolls a host, and a new
// row would move the hosts baseline.

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

async function enrollStockedHost(req: APIRequestContext, name: string) {
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
      services: [
        { unit: 'nginx.service', active_state: 'active', sub_state: 'running' },
        { unit: 'postgresql.service', active_state: 'active', sub_state: 'running' },
        { unit: 'cups.service', active_state: 'inactive', sub_state: 'dead' },
      ],
      containers: [
        { id: 'c1', name: 'grafana', image: 'grafana/grafana', state: 'running', health: '' },
        { id: 'c2', name: 'postgres-db', image: 'postgres:16', state: 'running', health: '' },
      ],
      cron_jobs: [
        { name: 'nightly-backup', schedule: '0 3 * * *' },
        { name: 'log-rotate', schedule: '0 0 * * 0' },
      ],
      processes: [
        { pid: 101, user: 'root', command: '/usr/sbin/nginx', cpu_pct: 3, mem_rss: 1000 },
        { pid: 202, user: 'postgres', command: '/usr/lib/postgresql/16/bin/postgres', cpu_pct: 8, mem_rss: 2000 },
      ],
      metrics: { cpu_pct: 5, mem_used: 500, mem_total: 1000, disk_used: 16, disk_total: 30 },
    },
  });
  expect(push.status()).toBe(200);
  return body.host.id as string;
}

// The heading sits in a row beside its search box, and that row's parent holds
// the list: two steps up from the h2 is the section.
function section(page: Page, title: string) {
  return page.locator(`h2:text-is("${title}")`).locator('xpath=../..');
}

test('each inventory list filters from its own heading', async ({ page }) => {
  await login(page);
  const hostID = await enrollStockedHost(page.request, 'e2e-inventory-search');
  await page.goto(`/hosts/${hostID}`);

  const units = section(page, 'Systemd units');
  await expect(units).toContainText('Systemd units (3)');

  await units.getByRole('searchbox', { name: 'Search systemd units…' }).fill('postgres');
  await expect(units).toContainText('Systemd units (1)');
  await expect(units).toContainText('postgresql.service');
  await expect(units).not.toContainText('nginx.service');

  // The containers list is untouched by the units search: one box, one list.
  const containers = section(page, 'Containers');
  await expect(containers).toContainText('Containers (2)');

  await containers.getByRole('searchbox', { name: 'Search containers…' }).fill('grafana');
  await expect(containers).toContainText('Containers (1)');
  await expect(containers).not.toContainText('postgres-db');
  await expect(units).toContainText('Systemd units (1)');

  const cron = section(page, 'Cron jobs');
  await cron.getByRole('searchbox', { name: 'Search cron jobs…' }).fill('backup');
  await expect(cron).toContainText('Cron jobs (1)');
  await expect(cron).toContainText('nightly-backup');
});

test('an inventory search with no match says so', async ({ page }) => {
  await login(page);
  const hostID = await enrollStockedHost(page.request, 'e2e-inventory-nomatch');
  await page.goto(`/hosts/${hostID}`);

  const units = section(page, 'Systemd units');
  await units.getByRole('searchbox', { name: 'Search systemd units…' }).fill('nosuchunit');
  await expect(units).toContainText('Nothing here matches the search.');
  await expect(units).toContainText('Systemd units (0)');
});

test('the process table filters by command, user and pid', async ({ page }) => {
  await login(page);
  const hostID = await enrollStockedHost(page.request, 'e2e-process-search');
  await page.goto(`/hosts/${hostID}`);

  const table = page.locator('table').filter({ hasText: '/usr/sbin/nginx' });
  await expect(table.locator('tbody tr')).toHaveCount(2);

  const box = page.getByRole('searchbox', { name: 'Search processes…' });
  await box.fill('nginx');
  await expect(table.locator('tbody tr')).toHaveCount(1);

  await box.fill('postgres');
  await expect(page.locator('table').filter({ hasText: 'postgresql' }).locator('tbody tr')).toHaveCount(1);

  await box.fill('101');
  await expect(page.locator('table').filter({ hasText: '/usr/sbin/nginx' }).locator('tbody tr')).toHaveCount(1);

  await box.fill('nosuchprocess');
  await expect(page.getByText('No process matches the search.')).toBeVisible();
});
