import { test, expect } from '@playwright/test';

// A primary button without a border is 2px shorter than an outlined one, and
// any row mixing them sits crooked. That shipped, so it gets a test — and one
// that sweeps every view rather than the row it was noticed on, since the cause
// is shared and the next occurrence will be somewhere else.
//
// Depends on core.spec creating admin@example.com first in the serial run.

const VIEWS = ['/', '/services', '/collections', '/hosts', '/admin/users', '/admin/groups', '/admin/webhooks', '/admin/alerts', '/admin/server', '/admin/settings', '/profile'];

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

    // Group by parent, so only buttons that actually sit side by side are
    // compared — a button in a card header and one in a footer may differ.
    const groups = await page.evaluate(() => {
      const byParent = new Map<Element, HTMLElement[]>();
      for (const b of document.querySelectorAll<HTMLElement>('button')) {
        if (b.offsetParent === null) {
          continue;
        }
        const p = b.parentElement;
        if (!p) {
          continue;
        }
        const list = byParent.get(p) ?? [];
        list.push(b);
        byParent.set(p, list);
      }
      const out: { labels: string[]; heights: number[] }[] = [];
      for (const buttons of byParent.values()) {
        if (buttons.length < 2) {
          continue;
        }
        // Only compare a genuine horizontal row: buttons stacked vertically are
        // free to differ, and a wrapped row would report a false mismatch.
        const tops = buttons.map((b) => Math.round(b.getBoundingClientRect().top));
        if (new Set(tops).size !== 1) {
          continue;
        }
        out.push({
          labels: buttons.map((b) => (b.textContent ?? '').trim().slice(0, 24)),
          heights: buttons.map((b) => Math.round(b.getBoundingClientRect().height)),
        });
      }
      return out;
    });

    for (const g of groups) {
      const unique = new Set(g.heights);
      if (unique.size > 1) {
        offenders.push(`${view}: ${g.labels.map((l, i) => `"${l}"=${g.heights[i]}px`).join(', ')}`);
      }
    }
  }

  expect(offenders, `buttons in the same row differ in height:\n${offenders.join('\n')}`).toEqual([]);
});
