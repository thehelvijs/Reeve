export interface DiskUsage {
  mount: string;
  device: string;
  fs_type: string;
  used: number;
  total: number;
}

// usedPct is a filesystem's fullness, 0 for one that reported no capacity.
export function usedPct(d: DiskUsage): number {
  if (d.total <= 0) {
    return 0;
  }
  return (d.used / d.total) * 100;
}

// sortDisks puts the fullest filesystem first, since the reason to read this
// list is finding the one about to run out, and that is rarely the root. Ties
// break on mount so the order does not shuffle between polls.
export function sortDisks(disks: DiskUsage[]): DiskUsage[] {
  const copy = [...disks];
  copy.sort((a, b) => {
    const diff = usedPct(b) - usedPct(a);
    if (diff !== 0) {
      return diff;
    }
    return a.mount.localeCompare(b.mount);
  });
  return copy;
}
