import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

async function enrollHost(req: APIRequestContext, name: string) {
  const res = await req.post('/api/v1/admin/hosts', { data: { name } });
  expect(res.status()).toBe(201);
  const body = await res.json();
  return { id: body.host.id as string, token: body.enroll_token as string };
}

// Reports an address over the real ingest path, the way the agent does.
async function reportAddress(req: APIRequestContext, token: string, ip: string) {
  const res = await req.post('/api/v1/ingest', {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      agent_version: '9.9.9',
      ip_address: ip,
      sent_at: new Date().toISOString(),
      metrics: {},
    },
  });
  expect(res.status()).toBe(200);
}

async function createTool(req: APIRequestContext, data: Record<string, unknown>) {
  const res = await req.post('/api/v1/tools', { data });
  expect(res.status()).toBe(201);
  return res.json();
}

// The whole point: one link that keeps working when the host's lease changes.
test('a short link follows the host address across a lease change', async ({ page }) => {
  await login(page);
  const request = page.request;
  const host = await enrollHost(request, 'shortlink-host');
  await reportAddress(request, host.token, '192.168.77.10');

  const tool = await createTool(request, {
    name: 'Shortlink Grafana',
    source_type: 'manual',
    visibility: 'public',
    host_id: host.id,
    scheme: 'http',
    port: 3000,
  });
  expect(tool.slug).toBe('shortlink-grafana');

  const first = await request.get(`/go/${tool.slug}`, { maxRedirects: 0 });
  expect(first.status()).toBe(302);
  expect(first.headers()['location']).toBe('http://192.168.77.10:3000');

  await reportAddress(request, host.token, '192.168.77.55');

  const second = await request.get(`/go/${tool.slug}`, { maxRedirects: 0 });
  expect(second.headers()['location']).toBe('http://192.168.77.55:3000');

  const json = await (await request.get(`/api/v1/endpoints/${tool.slug}`)).json();
  expect(json.source).toBe('host');
  expect(json.host_ip).toBe('192.168.77.55');
});

// A typed address is a deliberate choice and must survive the host reporting one.
test('a typed address is not overridden by the host address', async ({ page }) => {
  await login(page);
  const request = page.request;
  const host = await enrollHost(request, 'shortlink-host-pinned');
  await reportAddress(request, host.token, '192.168.77.30');

  const tool = await createTool(request, {
    name: 'Shortlink Pinned',
    source_type: 'manual',
    visibility: 'public',
    host_id: host.id,
    scheme: 'http',
    address: 'pinned.lan',
    port: 9000,
  });

  const res = await request.get(`/go/${tool.slug}`, { maxRedirects: 0 });
  expect(res.headers()['location']).toBe('http://pinned.lan:9000');
  const json = await (await request.get(`/api/v1/endpoints/${tool.slug}`)).json();
  expect(json.source).toBe('address');
});

test('the tool page shows the short link and points Open at it', async ({ page }) => {
  await login(page);
  const request = page.request;
  const host = await enrollHost(request, 'shortlink-host-2');
  await reportAddress(request, host.token, '192.168.77.20');
  const tool = await createTool(request, {
    name: 'Shortlink Paperless',
    source_type: 'manual',
    visibility: 'public',
    host_id: host.id,
    scheme: 'http',
    port: 8000,
  });

  await page.goto(`/catalog/${tool.id}`);
  await expect(page.getByText(`/go/${tool.slug}`)).toBeVisible();
  await expect(page.getByRole('link', { name: /open/i })).toHaveAttribute('href', `/go/${tool.slug}`);
});

// A tool nobody may see must not be discoverable through its short link.
test('a restricted tool 404s for an anonymous short link', async ({ page, browser }) => {
  await login(page);
  const request = page.request;
  const tool = await createTool(request, {
    name: 'Shortlink Secret',
    source_type: 'manual',
    visibility: 'restricted',
    scheme: 'http',
    address: '10.44.0.1',
  });

  const anon = await browser.newContext();
  const res = await anon.request.get(`http://127.0.0.1:5173/go/${tool.slug}`, { maxRedirects: 0 });
  expect(res.status()).toBe(404);
  await anon.close();
});
