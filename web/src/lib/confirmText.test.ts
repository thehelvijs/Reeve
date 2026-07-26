import { describe, expect, it } from 'vitest';
import { needsConfirm, rowConfirmation } from './confirmText';

describe('needsConfirm', () => {
  // Starting something that is down cannot lose anything; stop and restart take
  // whatever is using it down with them.
  it('gates stop and restart but not start', () => {
    expect(needsConfirm('stop')).toBe(true);
    expect(needsConfirm('restart')).toBe(true);
    expect(needsConfirm('start')).toBe(false);
  });
});

describe('rowConfirmation', () => {
  // These rows sit in a scrolling list, so "restart?" alone does not say what.
  it('names the target and the host', () => {
    const c = rowConfirmation('restart', 'plexmediaserver.service', 'helvishzima');
    expect(c.title).toContain('plexmediaserver.service');
    expect(c.body).toContain('plexmediaserver.service');
    expect(c.body).toContain('helvishzima');
    expect(c.confirmLabel).toBe('Restart');
  });

  it('says what stopping costs', () => {
    const c = rowConfirmation('stop', 'nginx', 'web-1');
    expect(c.confirmLabel).toBe('Stop');
    expect(c.body).toMatch(/goes down/);
  });
});
