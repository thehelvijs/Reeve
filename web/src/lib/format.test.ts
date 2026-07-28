import { describe, expect, it } from 'vitest';
import { fmtBytes, fmtCount, fmtRate, fmtUptime } from './format';

describe('fmtBytes', () => {
  it('stops climbing units at the largest one it knows', () => {
    expect(fmtBytes(0)).toBe('0B');
    expect(fmtBytes(1023)).toBe('1023B');
    expect(fmtBytes(1024)).toBe('1KB');
    expect(fmtBytes(1024 ** 3)).toBe('1GB');
    expect(fmtBytes(1024 ** 4)).toBe('1TB');
    // A petabyte has no unit in the table, so it reads as thousands of TB
    // rather than falling off the end as undefined.
    expect(fmtBytes(1024 ** 5)).toBe('1024TB');
  });

  it('rounds rather than truncating', () => {
    expect(fmtBytes(1024 * 1.6)).toBe('2KB');
  });
});

describe('fmtRate', () => {
  it('reads as bytes over a second', () => {
    expect(fmtRate(0)).toBe('0B/s');
    expect(fmtRate(1024)).toBe('1KB/s');
    expect(fmtRate(1024 ** 2 * 12)).toBe('12MB/s');
  });
});

describe('fmtCount', () => {
  it('pluralises only away from one', () => {
    expect(fmtCount(1, 'host')).toBe('1 host');
    expect(fmtCount(0, 'host')).toBe('0 hosts');
    expect(fmtCount(2, 'host')).toBe('2 hosts');
  });
});

describe('fmtUptime', () => {
  it('splits seconds into days, hours and minutes', () => {
    expect(fmtUptime(0)).toBe('0d 0h 0m');
    expect(fmtUptime(59)).toBe('0d 0h 0m');
    expect(fmtUptime(90)).toBe('0d 0h 1m');
    expect(fmtUptime(86400 + 3600 + 60)).toBe('1d 1h 1m');
    // Each unit is the remainder of the one above, so hours never exceed 23.
    expect(fmtUptime(86400 * 3 - 1)).toBe('2d 23h 59m');
  });
});
