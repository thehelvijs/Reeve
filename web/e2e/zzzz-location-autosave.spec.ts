import { test, expect, type Page } from '@playwright/test';

// The geocoder is stubbed at the browser, so this spec never leaves the machine
// and never depends on Photon being up. The server-side proxy is covered by
// TestGeocodeProxiesAndLabels.
// Two hits on purpose: the second row hangs over the map, and a suggestion the
// map paints over is unreadable and unclickable.
const HITS = [
  { label: 'Brivibas gatve 214, Riga, Latvia', lat: 57.0, lon: 24.15 },
  { label: 'Brivibas iela 32, Riga, Latvia', lat: 56.96, lon: 24.12 },
];
const ADDRESS = HITS[1];

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

async function locationHost(page: Page) {
  const res = await page.request.post('/api/admin/hosts', { data: { name: 'e2e-location' } });
  expect(res.ok(), 'could not create the host to locate').toBe(true);
  const body = await res.json();
  return body.host.id as string;
}

// The zoom a tile was cut for is in its URL, which is the only handle the page
// gives on the map's view.
async function tileZoom(page: Page) {
  const src = await page.locator('.leaflet-tile').first().getAttribute('src');
  const z = src?.match(/\/(\d+)\/\d+\/\d+[@.]/);
  expect(z, `no zoom in the tile URL: ${src}`).not.toBeNull();
  return Number(z?.[1]);
}

test.describe('location picker', () => {
  test('suggests an address and autosaves the pick, with no save button', async ({ page }) => {
    await login(page);
    const id = await locationHost(page);

    let asked = '';
    await page.route('**/api/admin/geocode**', async (route) => {
      asked = new URL(route.request().url()).searchParams.get('q') ?? '';
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(HITS) });
    });

    await page.goto(`/hosts/${id}?tab=settings`);
    const field = page.locator('input[placeholder^="Search a city"]');
    await expect(field).toBeVisible();
    await expect(page.locator('button:has-text("Save location")')).toHaveCount(0);

    const saved = page.waitForResponse(
      (r) => r.url().includes(`/api/admin/hosts/${id}`) && r.request().method() === 'PATCH',
    );
    await field.fill('Brivibas iela 32');
    const option = page.locator(`button:has-text("${ADDRESS.label}")`);
    await expect(option).toBeVisible();
    expect(asked, 'the typed text never reached the geocoder').toBe('Brivibas iela 32');

    // The row below the first one hangs over the map. Paint order, not just
    // visibility, decides whether it can be read and clicked.
    const onTop = await option.evaluate((el) => {
      const r = el.getBoundingClientRect();
      const hit = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
      return el.contains(hit) || el === hit;
    });
    expect(onTop, 'the map paints over the suggestion list').toBe(true);
    await option.click();

    const patch = await saved;
    expect(patch.request().postDataJSON()).toMatchObject({
      physical_location: ADDRESS.label,
      latitude: ADDRESS.lat,
      longitude: ADDRESS.lon,
    });
    await expect(page.locator('text=Saved.')).toBeVisible();

    // The pin survives a reload, which is the only proof the write landed.
    await page.reload();
    await expect(page.locator('input[placeholder^="Search a city"]')).toHaveValue(ADDRESS.label);
    await expect(page.locator(`text=${ADDRESS.lat.toFixed(3)}, ${ADDRESS.lon.toFixed(3)}`)).toBeVisible();

    expect((await page.request.delete(`/api/admin/hosts/${id}`)).ok()).toBe(true);
  });

  test('autosaves a name the geocoder does not know', async ({ page }) => {
    await login(page);
    const id = await locationHost(page);
    await page.route('**/api/admin/geocode**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: '[]' }),
    );

    await page.goto(`/hosts/${id}?tab=settings`);
    const field = page.locator('input[placeholder^="Search a city"]');
    const saved = page.waitForResponse(
      (r) => r.url().includes(`/api/admin/hosts/${id}`) && r.request().method() === 'PATCH',
    );
    await field.fill('rack 3, basement');
    const patch = await saved;
    expect(patch.request().postDataJSON()).toMatchObject({
      physical_location: 'rack 3, basement',
      latitude: null,
      longitude: null,
    });
    await expect(page.locator('text=Saved.')).toBeVisible();

    expect((await page.request.delete(`/api/admin/hosts/${id}`)).ok()).toBe(true);
  });

  test('a chosen place moves the map, a dropped pin leaves it where it is', async ({ page }) => {
    await login(page);
    const id = await locationHost(page);
    await page.route('**/api/admin/geocode**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(HITS) }),
    );

    await page.goto(`/hosts/${id}?tab=settings`);
    const field = page.locator('input[placeholder^="Search a city"]');
    await expect(field).toBeVisible();
    // The world view the map opens on.
    await expect.poll(() => tileZoom(page)).toBe(2);

    await field.fill('Brivibas iela 32');
    await page.locator(`button:has-text("${ADDRESS.label}")`).click();
    // A place chosen by name is worth flying to: the city fills the frame.
    await expect.poll(() => tileZoom(page)).toBe(11);
    await expect(page.locator('text=Saved.')).toBeVisible();

    const map = page.locator('.leaflet-container');
    await map.scrollIntoViewIfNeeded();
    // Zoom past the fitted level, or a view that snapped back would land on the
    // same number and this would prove nothing.
    await page.locator('.leaflet-control-zoom-in').click();
    await expect.poll(() => tileZoom(page)).toBe(12);
    const box = (await map.boundingBox())!;
    const dropped = page.waitForResponse(
      (r) => r.url().includes(`/api/admin/hosts/${id}`) && r.request().method() === 'PATCH',
    );
    await map.click({ position: { x: box.width * 0.7, y: box.height * 0.3 } });
    const patch = await dropped;
    const body = patch.request().postDataJSON();
    expect(body.latitude, 'the click dropped no pin').not.toBe(ADDRESS.lat);
    expect(body.physical_location, 'the click rewrote the name').toBe(ADDRESS.label);
    // What this test exists for: the pin moved and the view did not.
    expect(await tileZoom(page)).toBe(12);

    expect((await page.request.delete(`/api/admin/hosts/${id}`)).ok()).toBe(true);
  });
});
