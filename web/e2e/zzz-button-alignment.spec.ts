import { test, expect } from '@playwright/test';

// A control that is button-shaped but not the Button component ends up a few
// pixels short of the one beside it, and any row mixing them sits crooked. That
// shipped twice, so it gets a test — and one that sweeps every view rather than
// the row it was noticed on, since the cause is shared.
//
// The first version of this test grouped by identical `top`, which is exactly
// wrong: a shorter control in an items-center row IS vertically offset, so the
// filter excluded the mismatches it was written to find and could never fail.
// Rows are matched by vertical overlap instead.
//
// Depends on core.spec creating admin@example.com first in the serial run.

const VIEWS = [
  '/',
  '/services',
  '/services/new',
  '/collections',
  '/hosts',
  '/requests',
  '/profile',
  '/admin/users',
  '/admin/groups',
  '/admin/webhooks',
  '/admin/alerts',
  '/admin/audit',
  '/admin/settings',
  '/admin/server',
];

// Anything that reads as a button: real buttons, and links or labels wearing the
// button box. Pills are excluded — a pill is a different shape on purpose, and
// a row may legitimately mix a pill with a button.
const SELECTOR = 'button, a[class*="rounded-button"], label[class*="rounded-button"]';

async function mismatches(page: import('@playwright/test').Page, label: string) {
  return page.evaluate((sel) => {
    const byParent = new Map<Element, { label: string; h: number; top: number; cls: string }[]>();
    for (const el of document.querySelectorAll<HTMLElement>(sel)) {
      if (el.offsetParent === null || el.className.includes('rounded-pill')) {
        continue;
      }
      const r = el.getBoundingClientRect();
      if (r.height === 0) {
        continue;
      }
      const parent = el.parentElement;
      if (!parent) {
        continue;
      }
      const list = byParent.get(parent) ?? [];
      list.push({
        label: (el.textContent ?? '').trim().slice(0, 24),
        h: Math.round(r.height),
        top: r.top,
        cls: el.className.slice(0, 60),
      });
      byParent.set(parent, list);
    }
    const out: string[] = [];
    for (const items of byParent.values()) {
      if (items.length < 2 || new Set(items.map((i) => i.h)).size === 1) {
        continue;
      }
      // A genuine visual row: two of them overlap vertically. Stacked controls
      // are free to differ.
      const sameRow = items.some((a) =>
        items.some((b) => a !== b && a.top < b.top + b.h && b.top < a.top + a.h),
      );
      if (!sameRow) {
        continue;
      }
      out.push(items.map((i) => `"${i.label}"=${i.h}px [${i.cls}]`).join(' vs '));
    }
    return out;
  }, SELECTOR).then((rows) => rows.map((r) => `${label}: ${r}`));
}

test('buttons sharing a row are the same height', async ({ page }) => {
  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@example.com');
  await page.fill('input[type=password]', 'password123');
  await page.click('button[type=submit]');
  await expect(page).toHaveURL('/');

  const offenders: string[] = [];
  for (const view of VIEWS) {
    await page.goto(view);
    await page.waitForTimeout(400);
    offenders.push(...(await mismatches(page, view)));
  }

  // Modals are half the buttons in the app and no screenshot sweep opens them
  // all, so the ones reachable from a header are checked here too.
  const modals: [string, string][] = [
    ['/hosts', 'Add host'],
    ['/hosts', 'How to add a host'],
    ['/services', 'Add for monitoring'],
    ['/collections', 'New collection'],
    ['/admin/groups', 'New group'],
  ];
  for (const [view, trigger] of modals) {
    await page.goto(view);
    await page.waitForTimeout(300);
    const button = page.locator(`button:text-is("${trigger}")`).first();
    if ((await button.count()) === 0) {
      continue;
    }
    await button.click();
    await expect(page.locator('[role=dialog]')).toBeVisible();
    await page.waitForTimeout(300);
    offenders.push(...(await mismatches(page, `${view} → ${trigger}`)));
    await page.keyboard.press('Escape');
  }

  expect(offenders, `controls in the same row differ in height:\n${offenders.join('\n')}`).toEqual([]);
});
