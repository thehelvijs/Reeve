import { describe, expect, it } from 'vitest';
import { sortDisks, usedPct, type DiskUsage } from './disks';

const d = (mount: string, used: number, total: number): DiskUsage => ({
  mount,
  device: '/dev/x',
  fs_type: 'ext4',
  used,
  total,
});

describe('usedPct', () => {
  it('is the fill fraction as a percentage', () => {
    expect(usedPct(d('/', 50, 200))).toBe(25);
  });

  // A filesystem that reported nothing must not become NaN or Infinity in a
  // width style, which renders as a bar of unpredictable length.
  it('is zero for a filesystem with no capacity', () => {
    expect(usedPct(d('/', 0, 0))).toBe(0);
  });
});

describe('sortDisks', () => {
  // The reason to read the list is finding the disk about to run out, and on a
  // real machine that is rarely the root filesystem.
  it('puts the fullest filesystem first', () => {
    const got = sortDisks([d('/', 10, 100), d('/mnt/media', 95, 100), d('/boot', 40, 100)]);
    expect(got.map((x) => x.mount)).toEqual(['/mnt/media', '/boot', '/']);
  });

  it('breaks ties on mount so the order is stable between polls', () => {
    const got = sortDisks([d('/zzz', 1, 2), d('/aaa', 1, 2)]);
    expect(got.map((x) => x.mount)).toEqual(['/aaa', '/zzz']);
  });

  it('does not mutate the input', () => {
    const disks = [d('/', 1, 100), d('/full', 99, 100)];
    sortDisks(disks);
    expect(disks.map((x) => x.mount)).toEqual(['/', '/full']);
  });
});
