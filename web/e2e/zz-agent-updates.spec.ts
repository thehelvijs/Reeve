import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

const SERVER_VERSION = '9.9.9';

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

async function createHost(req: APIRequestContext, name: string) {
  const res = await req.post('/api/v1/admin/hosts', { data: { name } });
  expect(res.status()).toBe(201);
  const body = await res.json();
  return { id: body.host.id as string, token: body.enroll_token as string };
}

// Drives the real ingest protocol rather than mocking it, so the ack and the rollout state machine are exercised exactly as a live agent would.
async function push(
  req: APIRequestContext,
  token: string,
  agentVersion: string,
  vetoed = false,
): Promise<{ check_now: boolean }> {
  const res = await req.post('/api/v1/ingest', {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      protocol_version: 2,
      agent_version: agentVersion,
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
  const res = await req.put('/api/v1/admin/settings', { data: { agent_update: patch } });
  expect(res.status()).toBe(200);
}

// Each row is an <a> wrapping name+pills; a second, textless <a> wraps only the chevron. Scoped to the list because the paused-rollout banner also links stalled hosts by name.
function hostRow(page: Page, name: string) {
  return page.locator('#host-list a', { hasText: name });
}

function pausedBanner(page: Page) {
  return page.locator('p', { hasText: 'Rollout paused' });
}

test('a current agent reads as up to date and an old one as outdated', async ({ page }) => {
  await login(page);
  const req = page.request;
  await setFleetPolicy(req, { enabled: true, concurrency: 10, stall_secs: 900 });

  const current = await createHost(req, 'e2e-current');
  const behind = await createHost(req, 'e2e-behind');

  const currentAck = await push(req, current.token, SERVER_VERSION);
  expect(currentAck.check_now).toBe(false);

  const behindAck = await push(req, behind.token, '0.0.1');
  expect(behindAck.check_now).toBe(true);

  await page.goto('/hosts');
  const currentRow = hostRow(page, 'e2e-current');
  const behindRow = hostRow(page, 'e2e-behind');

  // Each assertion below targets the exact row it names, so it stands on its own regardless of what else runs first.
  await expect(currentRow).toContainText(`agent ${SERVER_VERSION}`);
  await expect(behindRow).toContainText('agent 0.0.1');

  // check_now=true means the server already granted the behind host a slot in the same push, so it reads "updating", not "outdated" (see updateStateFor in server/agent_update.go).
  await expect(behindRow.getByText('updating', { exact: true })).toBeVisible();
  await expect(currentRow.getByText('updating', { exact: true })).toHaveCount(0);
  await expect(currentRow.getByText('outdated', { exact: true })).toHaveCount(0);
  await expect(page.locator('text=server on 9.9.9')).toBeVisible();

  // Release the slot the outdated push claimed so it doesn't count against a later test's tighter concurrency cap.
  await push(req, behind.token, SERVER_VERSION);
});

test('a host that vetoes locally is never told to update', async ({ page }) => {
  await login(page);
  const req = page.request;
  await setFleetPolicy(req, { enabled: true, concurrency: 10, stall_secs: 900 });

  const vetoed = await createHost(req, 'e2e-vetoed');
  const ack = await push(req, vetoed.token, '0.0.1', true);
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

  expect((await push(req, first.token, '0.0.1')).check_now).toBe(true);
  expect((await push(req, second.token, '0.0.1')).check_now).toBe(false);

  // The capped host never got a slot, so it stays outdated (UpdateStartedAt still null) — the feature's headline state.
  await page.goto('/hosts');
  await expect(hostRow(page, 'e2e-cap-2').getByText('outdated', { exact: true })).toBeVisible();

  // Free the slot and restore a sane concurrency so later tests start clean even if this spec is ever run as a standalone subset.
  await push(req, first.token, SERVER_VERSION);
  await setFleetPolicy(req, { enabled: true, concurrency: 10, stall_secs: 900 });
});

test('a stalled host pauses the rollout until it is resumed', async ({ page }) => {
  await login(page);
  const req = page.request;
  await setFleetPolicy(req, { enabled: true, concurrency: 5, stall_secs: 1 });

  const canary = await createHost(req, 'e2e-canary');
  expect((await push(req, canary.token, '0.0.1')).check_now).toBe(true);

  await page.waitForTimeout(2000);
  expect((await push(req, canary.token, '0.0.1')).check_now).toBe(false);

  await page.goto('/hosts');
  const canaryRow = hostRow(page, 'e2e-canary');
  await expect(canaryRow.getByText('update stalled', { exact: true })).toBeVisible();
  await expect(pausedBanner(page)).toBeVisible();

  await page.getByRole('button', { name: 'Resume rollout' }).click();
  await expect(pausedBanner(page)).toHaveCount(0);

  await setFleetPolicy(req, { enabled: true, concurrency: 5, stall_secs: 900 });
  await push(req, canary.token, SERVER_VERSION);
});

test('a per-host policy of off disables updates for that host alone', async ({ page }) => {
  await login(page);
  const req = page.request;
  await setFleetPolicy(req, { enabled: true, concurrency: 10, stall_secs: 900 });

  const pinned = await createHost(req, 'e2e-pinned-off');
  await push(req, pinned.token, '0.0.1');

  await page.goto(`/hosts/${pinned.id}`);
  await expect(page.getByText('Agent', { exact: true })).toBeVisible();
  await page.selectOption('select', 'off');
  // Wait for the PUT the select triggers to land before the next push relies on it.
  await expect(page.getByText('updates off', { exact: true })).toBeVisible();

  const ack = await push(req, pinned.token, '0.0.1');
  expect(ack.check_now).toBe(false);

  await page.reload();
  await expect(page.getByText('updates off', { exact: true })).toBeVisible();

  // A second, unpinned host under the same fleet policy still gets a slot — proves the policy scopes to this host alone.
  const control = await createHost(req, 'e2e-not-pinned');
  expect((await push(req, control.token, '0.0.1')).check_now).toBe(true);
  await push(req, control.token, SERVER_VERSION);

  // Setting the policy off released the slot the first push claimed, so the host stays out of the rollup's stalled set no matter how long it sits on the old version. Asserted, not worked around: a dangling row here used to read as a fleet-wide stall.
  const rollup = await (await req.get('/api/v1/admin/agent-updates')).json();
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
  expect((await push(req, broken.token, '0.0.1')).check_now).toBe(true);
  await page.waitForTimeout(2000);
  await push(req, broken.token, '0.0.1');

  await page.goto('/hosts');
  await expect(pausedBanner(page)).toBeVisible();

  // The banner names the host as a link, which is the only signposted route to the permanent fix.
  await pausedBanner(page).getByRole('link', { name: 'e2e-never-updates' }).click();
  await expect(page).toHaveURL(new RegExp(`/hosts/${broken.id}$`));
  await page.selectOption('select', 'off');
  await expect(page.getByText('updates off', { exact: true })).toBeVisible();

  await page.goto('/hosts');
  await expect(pausedBanner(page)).toHaveCount(0);

  const rollup = await (await req.get('/api/v1/admin/agent-updates')).json();
  expect(rollup.paused).toBe(false);
  expect(rollup.counts.stalled).toBe(0);

  await setFleetPolicy(req, { enabled: true, concurrency: 5, stall_secs: 900 });
});
