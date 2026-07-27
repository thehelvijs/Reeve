import AxeBuilder from '@axe-core/playwright';
import { test, expect, type Page } from '@playwright/test';

// DESIGN.md commits to contrast and visible focus rings; nothing verified either
// until now. Axe checks what a machine can: contrast ratios, control names,
// landmark structure, label association.
//
// Every view is scanned in both themes. Two token maps mean two sets of contrast
// ratios, and a step that reads on one canvas can fail on the other — that is the
// whole reason this file loops.
//
// Scoped to WCAG 2 A/AA, which is the level the design system actually claims.
// Rules outside that (best-practice heuristics) are advisory and would turn this
// into a nag rather than a gate.
const TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'];

const THEMES = ['dark', 'light'] as const;

// Written before any navigation, so public/theme-boot.js applies it on the first
// paint of every page the test opens.
async function forceTheme(page: Page, theme: 'dark' | 'light') {
  await page.addInitScript((t) => {
    window.localStorage.setItem('reeve-theme', t);
  }, theme);
}

async function scan(page: Page) {
  return new AxeBuilder({ page })
    // Leaflet builds its own DOM: its tile <img>s carry no alt and its zoom
    // links are its own markup, so violations there are the library's, not
    // ours, and we cannot fix them without forking it.
    .exclude('.leaflet-container')
    .withTags(TAGS)
    .analyze();
}

// Prints file:line-style detail on failure — an id and a count alone is not
// enough to act on.
function report(violations: Awaited<ReturnType<typeof scan>>['violations']): string {
  return violations
    .map((v) => `${v.id} (${v.impact}): ${v.help}\n    ${v.nodes.map((n) => n.target.join(' ')).join('\n    ')}`)
    .join('\n');
}

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');
}

for (const theme of THEMES) {
  test.describe(`${theme} theme`, () => {
    test.beforeEach(async ({ page }) => {
      await forceTheme(page, theme);
    });

    test('anonymous portal has no accessibility violations', async ({ page, context }) => {
      await context.clearCookies();
      await page.route('**/basemaps.cartocdn.com/**', (route) => route.abort());
      await page.goto('/');
      await expect(page.locator('main')).toBeVisible();
      await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
      const { violations } = await scan(page);
      expect(report(violations), report(violations)).toBe('');
    });

    test('sign-in page has no accessibility violations', async ({ page, context }) => {
      await context.clearCookies();
      await page.goto('/login');
      await expect(page.locator('input[type=email]')).toBeVisible();
      const { violations } = await scan(page);
      expect(report(violations), report(violations)).toBe('');
    });

    test.describe('authed app', () => {
      test.beforeEach(async ({ page }) => {
        await page.route('**/basemaps.cartocdn.com/**', (route) => route.abort());
        await login(page);
      });

      for (const [name, path] of Object.entries({
        dashboard: '/dashboard',
        services: '/services',
        'services-new': '/services/new',
        hosts: '/hosts',
        profile: '/profile',
        'admin-users': '/admin/users',
        'admin-alerts': '/admin/alerts',
        'admin-settings': '/admin/settings',
        // Excluded from the visual sweep as a live machine readout, so this is
        // the only place it is checked at all.
        'admin-server': '/admin/server',
      })) {
        test(`${name} has no accessibility violations`, async ({ page }) => {
          await page.goto(path);
          await expect(page.locator('main')).toBeVisible();
          const { violations } = await scan(page);
          expect(report(violations), report(violations)).toBe('');
        });
      }
    });

    // A modal is the easiest thing to get wrong: it has to be announced as a
    // dialog and its controls have to be reachable and named.
    test('the service modal has no accessibility violations', async ({ page, context }) => {
      await context.clearCookies();
      await page.goto('/');
      await page.click('text=PortalGrafana');
      await expect(page.getByRole('dialog', { name: 'Service' })).toBeVisible();
      const { violations } = await scan(page);
      expect(report(violations), report(violations)).toBe('');
    });
  });
}
