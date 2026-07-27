import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

const SERVER_VERSION = '9.9.9';

// The server decides "current" by comparing the sha256 an agent reports to the
// checksum of the build it publishes; version strings decide nothing. Read the
// real published value rather than hardcoding one, so this stays true when the
// embedded binary is rebuilt.
const OLD_BUILD = '0'.repeat(64);

async function publishedChecksum(req: APIRequestContext): Promise<string> {
  const res = await req.get('/dl/agent-linux-amd64.sha256');
  expect(res.status()).toBe(200);
  const sum = (await res.text()).trim().split(/\s+/)[0];
  // An unproxied route answers with the SPA shell, and a 200 full of HTML then
  // reads as "this agent is on an unknown build" rather than as a broken test.
  expect(sum, 'expected a sha256, not the SPA shell').toMatch(/^[0-9a-f]{64}$/);
  return sum;
}

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

async function createHost(req: APIRequestContext, name: string) {
  const res = await req.post('/api/admin/hosts', { data: { name } });
  expect(res.status()).toBe(201);
  const body = await res.json();
  return { id: body.host.id as string, token: body.enroll_token as string };
}

// Drives the real ingest protocol rather than mocking it, so the ack and the rollout state machine are exercised exactly as a live agent would.
async function push(
  req: APIRequestContext,
  token: string,
  checksum: string,
  vetoed = false,
): Promise<{ check_now: boolean }> {
  const res = await req.post('/api/ingest', {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      agent_version: SERVER_VERSION,
      agent_checksum: checksum,
      auto_update_vetoed: vetoed,
      sent_at: new Date().toISOString(),
      metrics: {},
    },
  });
  expect(res.status()).toBe(200);
  return res.json();
}

async function setFleetPolicy(
  req: APIRequestContext,
  patch: { enabled: boolean; concurrency: number; stall_secs: number },
) {
  const res = await req.put('/api/admin/settings', { data: { agent_update: patch } });
  expect(res.status()).toBe(200);
}

// A row is a <tr> whose first cell links the host by name. Scoped to the list because the paused-rollout banner also links stalled hosts by name.
function hostRow(page: Page, name: string) {
  return page.locator('#host-list tr', { hasText: name });
}

function pausedBanner(page: Page) {
  return page.locator('p', { hasText: 'Rollout paused' });
}

test('a current agent reads as up to date and an old one as outdated', async ({ page }) => {
  await login(page);
  const req = page.request;
  await setFleetPolicy(req, { enabled: true, concurrency: 10, stall_secs: 900 });

  const current_sum = await publishedChecksum(req);
  const current = await createHost(req, 'e2e-current');
  const behind = await createHost(req, 'e2e-behind');

  const currentAck = await push(req, current.token, current_sum);
  expect(currentAck.check_now).toBe(false);

  const behindAck = await push(req, behind.token, OLD_BUILD);
  expect(behindAck.check_now).toBe(true);

  await page.goto('/hosts');
  const currentRow = hostRow(page, 'e2e-current');
  const behindRow = hostRow(page, 'e2e-behind');

  // Each assertion below targets the exact row it names, so it stands on its own regardless of what else runs first.
  // Both report the same version string in the Agent column: what separates them is the binary, which is the point.
  await expect(currentRow).toContainText(SERVER_VERSION);
  await expect(behindRow).toContainText(SERVER_VERSION);

  // check_now=true means the server already granted the behind host a slot in the same push, so it reads "updating", not "outdated" (see updateStateFor in server/agent_update.go).
  await expect(behindRow.getByText('updating', { exact: true })).toBeVisible();
  await expect(currentRow.getByText('updating', { exact: true })).toHaveCount(0);
  await expect(currentRow.getByText('outdated', { exact: true })).toHaveCount(0);
  await expect(page.locator('text=server on 9.9.9')).toBeVisible();

  // Release the slot the outdated push claimed so it doesn't count against a later test's tighter concurrency cap.
  await push(req, behind.token, current_sum);
});

// The Agent card used to offer Update on every host, including one already on
// the published build, where the only answer the server can give is a 409. The
// button now asks hostRowActions, the same predicate the host rows use.
test('the host page offers Update only when there is something to update to', async ({ page }) => {
  await login(page);
  const req = page.request;
  await setFleetPolicy(req, { enabled: true, concurrency: 10, stall_secs: 900 });
  const updateButton = page.getByRole('button', { name: 'Update now' });

  const current = await createHost(req, 'e2e-card-current');
  await push(req, current.token, await publishedChecksum(req));
  await page.goto(`/hosts/${current.id}`);
  await expect(page.getByText('up to date', { exact: true })).toBeVisible();
  await expect(updateButton).toHaveCount(0);

  const behind = await createHost(req, 'e2e-card-behind');
  await push(req, behind.token, OLD_BUILD);
  await page.goto(`/hosts/${behind.id}`);
  await expect(updateButton).toBeVisible();

  // Both hosts go again: the pixel baselines later in the run photograph the
  // host list, so a spec that leaves rows behind rewrites someone else's frame.
  await push(req, behind.token, await publishedChecksum(req));
  for (const h of [current, behind]) {
    expect((await req.delete(`/api/admin/hosts/${h.id}`)).status()).toBe(204);
  }
});

test('a host that vetoes locally is never told to update', async ({ page }) => {
  await login(page);
  const req = page.request;
  await setFleetPolicy(req, { enabled: true, concurrency: 10, stall_secs: 900 });

  const vetoed = await createHost(req, 'e2e-vetoed');
  const ack = await push(req, vetoed.token, OLD_BUILD, true);
  expect(ack.check_now).toBe(false);

  await page.goto('/hosts');
  const vetoedRow = hostRow(page, 'e2e-vetoed');
  await expect(vetoedRow).toBeVisible();
  await expect(vetoedRow.getByText('updates off', { exact: true })).toBeVisible();
});

test('concurrency caps how many hosts update at once', async ({ page }) => {
  await login(page);
  const req = page.request;
  await setFleetPolicy(req, { enabled: true, concurrency: 1, stall_secs: 900 });

  const first = await createHost(req, 'e2e-cap-1');
  const second = await createHost(req, 'e2e-cap-2');

  expect((await push(req, first.token, OLD_BUILD)).check_now).toBe(true);
  expect((await push(req, second.token, OLD_BUILD)).check_now).toBe(false);

  // The capped host never got a slot, so it stays outdated (UpdateStartedAt still null) — the feature's headline state.
  await page.goto('/hosts');
  await expect(hostRow(page, 'e2e-cap-2').getByText('outdated', { exact: true })).toBeVisible();

  // Free the slot and restore a sane concurrency so later tests start clean even if this spec is ever run as a standalone subset.
  await push(req, first.token, await publishedChecksum(req));
  await setFleetPolicy(req, { enabled: true, concurrency: 10, stall_secs: 900 });
});

test('a stalled host pauses the rollout until it is resumed', async ({ page }) => {
  await login(page);
  const req = page.request;
  await setFleetPolicy(req, { enabled: true, concurrency: 5, stall_secs: 1 });

  const canary = await createHost(req, 'e2e-canary');
  expect((await push(req, canary.token, OLD_BUILD)).check_now).toBe(true);

  await page.waitForTimeout(2000);
  expect((await push(req, canary.token, OLD_BUILD)).check_now).toBe(false);

  await page.goto('/hosts');
  const canaryRow = hostRow(page, 'e2e-canary');
  await expect(canaryRow.getByText('update stalled', { exact: true })).toBeVisible();
  await expect(pausedBanner(page)).toBeVisible();

  await page.getByRole('button', { name: 'Resume rollout' }).click();
  await expect(pausedBanner(page)).toHaveCount(0);

  await setFleetPolicy(req, { enabled: true, concurrency: 5, stall_secs: 900 });
  await push(req, canary.token, await publishedChecksum(req));
});

test('a per-host policy of off disables updates for that host alone', async ({ page }) => {
  await login(page);
  const req = page.request;
  await setFleetPolicy(req, { enabled: true, concurrency: 10, stall_secs: 900 });

  const pinned = await createHost(req, 'e2e-pinned-off');
  await push(req, pinned.token, OLD_BUILD);

  await page.goto(`/hosts/${pinned.id}`);
  await expect(page.getByRole('heading', { name: 'Agent updates' })).toBeVisible();
  await page.selectOption('select', 'off');
  // Wait for the PUT the select triggers to land before the next push relies on it.
  await expect(page.getByText('updates off', { exact: true })).toBeVisible();

  const ack = await push(req, pinned.token, OLD_BUILD);
  expect(ack.check_now).toBe(false);

  await page.reload();
  await expect(page.getByText('updates off', { exact: true })).toBeVisible();

  // A second, unpinned host under the same fleet policy still gets a slot — proves the policy scopes to this host alone.
  const control = await createHost(req, 'e2e-not-pinned');
  expect((await push(req, control.token, OLD_BUILD)).check_now).toBe(true);
  await push(req, control.token, await publishedChecksum(req));

  // Setting the policy off released the slot the first push claimed, so the host stays out of the rollup's stalled set no matter how long it sits on the old version. Asserted, not worked around: a dangling row here used to read as a fleet-wide stall.
  const rollup = await (await req.get('/api/admin/agent-updates')).json();
  expect(rollup.paused).toBe(false);
  expect(rollup.stalled).toEqual([]);
  expect(rollup.counts.stalled).toBe(0);
});

// A host taken out of the rollout must clear the pause, or Resume is the only escape and it hands the same broken host its slot straight back.
test('taking a stalled host out of the rollout clears the pause', async ({ page }) => {
  await login(page);
  const req = page.request;
  await setFleetPolicy(req, { enabled: true, concurrency: 5, stall_secs: 1 });

  const broken = await createHost(req, 'e2e-never-updates');
  expect((await push(req, broken.token, OLD_BUILD)).check_now).toBe(true);
  await page.waitForTimeout(2000);
  await push(req, broken.token, OLD_BUILD);

  await page.goto('/hosts');
  await expect(pausedBanner(page)).toBeVisible();

  // The banner names the host as a link, which is the only signposted route to the permanent fix.
  await pausedBanner(page).getByRole('link', { name: 'e2e-never-updates' }).click();
  await expect(page).toHaveURL(new RegExp(`/hosts/${broken.id}$`));
  await page.selectOption('select', 'off');
  await expect(page.getByText('updates off', { exact: true })).toBeVisible();

  await page.goto('/hosts');
  await expect(pausedBanner(page)).toHaveCount(0);

  const rollup = await (await req.get('/api/admin/agent-updates')).json();
  expect(rollup.paused).toBe(false);
  expect(rollup.counts.stalled).toBe(0);

  await setFleetPolicy(req, { enabled: true, concurrency: 5, stall_secs: 900 });
});
