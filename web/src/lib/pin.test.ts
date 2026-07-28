import { describe, expect, it } from 'vitest';
import { PIN_DEFAULT, pinFill, pinSVG } from './pin';

describe('pinFill', () => {
  it('keeps a #rrggbb colour', () => {
    expect(pinFill('#ff8800')).toBe('#ff8800');
    expect(pinFill('#FF8800')).toBe('#FF8800');
  });

  it('falls back to the accent for anything unset', () => {
    expect(pinFill()).toBe(PIN_DEFAULT);
    expect(pinFill('')).toBe(PIN_DEFAULT);
    expect(pinFill(null)).toBe(PIN_DEFAULT);
  });

  // The value is written into an inline fill, so a colour that is not a colour
  // must never reach the SVG.
  it('refuses anything that is not a plain hex colour', () => {
    for (const bad of [
      'red',
      '#fff',
      '#gggggg',
      '#ff8800; stroke: url(http://x/)',
      'url(#x)',
      '"/><script>alert(1)</script>',
    ]) {
      expect(pinFill(bad), bad).toBe(PIN_DEFAULT);
    }
  });
});

describe('pinSVG', () => {
  it('draws the teardrop in the given colour', () => {
    const svg = pinSVG('#ff8800');
    expect(svg).toContain('fill="#ff8800"');
    expect(svg).toContain('viewBox="0 0 24 32"');
  });

  it('carries no attacker-controlled text through', () => {
    expect(pinSVG('"><script>alert(1)</script>')).not.toContain('script');
  });
});
