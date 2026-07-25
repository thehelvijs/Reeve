export function fmtBytes(v: number): string {
  const u = ['B', 'KB', 'MB', 'GB', 'TB'];
  let i = 0;
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(0)}${u[i]}`;
}

export function fmtCount(n: number, singular: string): string {
  if (n === 1) {
    return `1 ${singular}`;
  }
  return `${n} ${singular}s`;
}

export function fmtUptime(secs: number): string {
  const d = Math.floor(secs / 86400);
  const h = Math.floor((secs % 86400) / 3600);
  const m = Math.floor((secs % 3600) / 60);
  return `${d}d ${h}h ${m}m`;
}
