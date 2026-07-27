import { describe, expect, it } from 'vitest';
import boot from '../../public/theme-boot.js?raw';
import { canvasFor, DEFAULT_THEME, otherTheme, readTheme, THEME_KEY } from './theme';

describe('readTheme', () => {
  it('defaults to dark', () => {
    expect(DEFAULT_THEME).toBe('dark');
    expect(readTheme(null)).toBe('dark');
    expect(readTheme('')).toBe('dark');
    expect(readTheme('system')).toBe('dark');
    expect(readTheme('DARK')).toBe('dark');
  });

  it('keeps a stored preference', () => {
    expect(readTheme('light')).toBe('light');
    expect(readTheme('dark')).toBe('dark');
  });
});

describe('otherTheme', () => {
  it('is what the toggle switches to', () => {
    expect(otherTheme('dark')).toBe('light');
    expect(otherTheme('light')).toBe('dark');
  });
});

// The boot script cannot import this module: it runs before the bundle, from a
// plain file. That makes the key and the default duplicated knowledge, and a
// drift means a stored preference is silently ignored on the next load.
describe('theme-boot.js agrees with this module', () => {
  it('reads the same storage key', () => {
    expect(boot).toContain(`'${THEME_KEY}'`);
  });

  it('falls back to the same default', () => {
    expect(boot).toContain(`var pref = '${DEFAULT_THEME}'`);
  });

  it('paints the same canvas behind each theme', () => {
    expect(boot).toContain(canvasFor('light'));
    expect(boot).toContain(canvasFor('dark'));
  });

  it('accepts exactly the themes this module accepts', () => {
    expect(boot).toContain("raw === 'light' || raw === 'dark'");
  });
});
