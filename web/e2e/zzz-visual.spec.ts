import { test, expect, type Locator, type Page } from '@playwright/test';

// Appearance regressions the behaviour specs cannot catch: a modal that renders
// behind a Leaflet map, a sidebar that overlaps content, a surface that loses
// its border. PLAYWRIGHT-CHECKLIST.md asks a human to look for these; these
// snapshots fail the build instead.
//
// Runs last (zzz-) so the instance holds everything the earlier specs created.
// Baselines live in zzz-visual.spec.ts-snapshots/ and are regenerated with
// `npm --prefix web run e2e -- --update-snapshots`.

// What is under test here is layout, not data. Anything whose text changes
// between runs is painted over first: clock-derived times, live host and process
// metrics, and the online/offline pills that flip with how long the suite took
// to get here. Masking is deliberate — the alternative is a loose diff
// threshold, which hides real breaks everywhere on the page instead of only
// where the data lives.
function volatile(page: Page) {
  return [
    // 7/26/2026, 7:21:20 PM — toLocaleString output, in any locale order.
    page.getByText(/\d{1,2}\/\d{1,2}\/\d{2,4}/),
    page.getByText(/\d+[dhms] ago/),
    page.getByText(/^\d+d \d+h \d+m$/),
    // 0%, 9GB / 15GB, 7/14 — metric readouts and rollup counts.
    page.getByText(/^\d+(\.\d+)?%$/),
    page.getByText(/^\d+(\.\d+)?[BKMGT]B( \/ \d+(\.\d+)?[BKMGT]B)?$/),
    page.getByText(/^\d+\/\d+$/),
    page.getByText(/^(online|offline|never)$/),
    // uPlot draws to a canvas from live samples; nothing about it is stable.
    page.locator('.uplot'),
  ];
}

// The basemap raster comes from basemaps.cartocdn.com (LeafletMap.tsx), so a
// snapshot including it would depend on a third party's tiles and on having a
// network at all. Blocked: what is under test is our own chrome, pins, controls
// and stacking order over the map, not someone else's imagery.
// A regex, not a glob: the tile URLs are a.basemaps.cartocdn.com through
// d.basemaps.cartocdn.com, and a `**/host/**` glob needs a slash before the
// host, so it silently matches nothing.
async function blockBasemap(page: Page) {
  await page.route(/basemaps\.cartocdn\.com/, (route) => route.abort());
}

async function settle(page: Page) {
  await page.waitForLoadState('networkidle');
  // uPlot canvases and Leaflet tiles finish a frame after the network goes
  // quiet, and web fonts swap in around then too.
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(400);
}

// A screenshot can miss an occluded modal: if the thing covering it is a dark
// map tile the pixel diff can stay under any sane threshold. Hit-testing cannot
// be fooled — whatever the browser would deliver a click to at the dialog's
// centre has to be the dialog itself or something inside it.
async function expectOnTop(page: Page, dialog: Locator) {
  const box = await dialog.boundingBox();
  expect(box, 'dialog has no box').not.toBeNull();
  const topmostIsInside = await page.evaluate(
    ({ x, y }) => {
      const el = document.elementFromPoint(x, y);
      return el ? !!el.closest('[role="dialog"]') : false;
    },
    { x: box!.x + box!.width / 2, y: box!.y + box!.height / 2 },
  );
  expect(topmostIsInside, 'something is painting over the dialog').toBe(true);
}

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

test.describe('portal, anonymous', () => {
  test.beforeEach(async ({ context, page }) => {
    await context.clearCookies();
    await blockBasemap(page);
  });

  test('list view', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('main')).toBeVisible();
    await settle(page);
    await expect(page).toHaveScreenshot('portal-list.png', {
      fullPage: true,
      mask: volatile(page),
    });
  });

  test('map view', async ({ page }) => {
    await page.goto('/');
    await page.click('button:has-text("Map")');
    await expect(page.locator('.leaflet-container')).toBeVisible();
    await settle(page);
    await expect(page).toHaveScreenshot('portal-map.png', { mask: volatile(page) });
  });

  test('graph view', async ({ page }) => {
    await page.goto('/');
    await page.click('button:has-text("Graph")');
    await expect(page.locator('main')).toBeVisible();
    await settle(page);
    await expect(page).toHaveScreenshot('portal-graph.png', { mask: volatile(page) });
  });

  test('service modal renders above the list', async ({ page }) => {
    await page.goto('/');
    await page.click('text=PortalGrafana');
    const dialog = page.getByRole('dialog', { name: 'Service' });
    await expect(dialog).toBeVisible();
    await settle(page);
    await expectOnTop(page, dialog);
    await expect(page).toHaveScreenshot('portal-modal-over-list.png', { mask: volatile(page) });
  });
});

// The regression this file exists for. Leaflet's own panes sit at z-index 200 to
// 700 and the map container is a stacking context of its own, so a modal that
// paints fine over the list can still land underneath the map. Nothing in the
// e2e data has coordinates, so this test gives a host some: without a pin there
// is nothing to click, and the test would skip while looking like coverage.
test.describe('portal map, anonymous', () => {
  test.beforeEach(async ({ page, context }) => {
    await blockBasemap(page);
    // The pin has to come from a host the anonymous portal can see, which is
    // /api/public/hosts — a host with no public tool never appears there, so
    // giving coordinates to just any host would produce no marker.
    const publicHosts = await (await page.request.get('/api/public/hosts')).json();
    expect(publicHosts.length, 'no public host to pin on the map').toBeGreaterThan(0);
    const target = publicHosts[0];

    await login(page);
    const res = await page.request.patch(`/api/admin/hosts/${target.id}`, {
      data: { physical_location: 'Riga, Latvia', latitude: 56.946, longitude: 24.106 },
    });
    expect(res.status()).toBe(200);
    await context.clearCookies();
  });

  test('host modal renders above the map', async ({ page }) => {
    await page.goto('/');
    await page.click('button:has-text("Map")');
    await expect(page.locator('.leaflet-container')).toBeVisible();
    const pin = page.locator('.leaflet-marker-icon').first();
    await expect(pin).toBeVisible();
    await pin.click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await settle(page);
    await expectOnTop(page, dialog);
    await expect(page).toHaveScreenshot('portal-modal-over-map.png', { mask: volatile(page) });
  });
});

test.describe('authed app', () => {
  test.beforeEach(async ({ page }) => {
    await blockBasemap(page);
    await login(page);
  });

  for (const [name, path] of Object.entries({
    dashboard: '/dashboard',
    services: '/services',
    'services-new': '/services/new',
    collections: '/collections',
    hosts: '/hosts',
    requests: '/requests',
    profile: '/profile',
    'admin-users': '/admin/users',
    'admin-groups': '/admin/groups',
    'admin-webhooks': '/admin/webhooks',
    'admin-alerts': '/admin/alerts',
    'admin-audit': '/admin/audit',
    'admin-settings': '/admin/settings',
    // /admin/server is deliberately absent: it is a live readout of the machine
    // the suite runs on — CPU, memory, disk and four uPlot charts of real
    // samples — so a pixel baseline would describe the runner, not the UI. It
    // is covered by the a11y sweep, whose structure is stable.
  })) {
    // Pages this change never touched are snapshotted too: a global CSS or
    // layout edit breaks the view nobody thought to open.
    test(name, async ({ page }) => {
      await page.goto(path);
      await expect(page.locator('main')).toBeVisible();
      await settle(page);
      await expect(page).toHaveScreenshot(`app-${name}.png`, {
        fullPage: true,
        mask: volatile(page),
      });
    });
  }
});
