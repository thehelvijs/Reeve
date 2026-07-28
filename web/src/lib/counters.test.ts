import { describe, expect, it } from 'vitest';
import { perSecond } from './counters';

describe('perSecond', () => {
  it('divides the counter delta by the elapsed seconds', () => {
    const xs = [0, 15, 30];
    const values = [1000, 2500, 2500 + 15 * 1024];
    expect(perSecond(values, xs)).toEqual([null, 100, 1024]);
  });

  it('has no rate for the first point', () => {
    expect(perSecond([500], [0])).toEqual([null]);
  });

  it('drops a counter that went backwards instead of drawing a negative spike', () => {
    // A reboot resets the counter, so the sample after it is smaller.
    const rates = perSecond([10_000, 200, 500], [0, 15, 30]);
    expect(rates).toEqual([null, null, 20]);
  });

  it('drops a pair with no elapsed time rather than dividing by zero', () => {
    const rates = perSecond([100, 200], [30, 30]);
    expect(rates).toEqual([null, null]);
    expect(rates.every((r) => r === null || Number.isFinite(r))).toBe(true);
  });

  it('reports a flat counter as no traffic, not as a gap', () => {
    expect(perSecond([777, 777, 777], [0, 15, 30])).toEqual([null, 0, 0]);
  });

  it('averages across a gap in the samples', () => {
    // The agent was offline for an hour; the traffic is spread over the gap.
    expect(perSecond([0, 3600], [0, 3600])).toEqual([null, 1]);
  });

  it('returns nothing for no points', () => {
    expect(perSecond([], [])).toEqual([]);
  });
});
