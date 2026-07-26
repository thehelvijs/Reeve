export interface ProcessSample {
  pid: number;
  user: string;
  command: string;
  cpu_pct: number;
  mem_rss: number;
}

export type ProcessSort = 'cpu' | 'mem';

// sortProcesses orders a snapshot by one dimension, descending, without
// mutating the caller's array. Ties break on pid so the row order is stable
// between polls and a table does not shuffle under the cursor.
export function sortProcesses(procs: ProcessSample[], by: ProcessSort): ProcessSample[] {
  const copy = [...procs];
  copy.sort((a, b) => {
    let diff = 0;
    if (by === 'cpu') {
      diff = b.cpu_pct - a.cpu_pct;
    } else {
      diff = b.mem_rss - a.mem_rss;
    }
    if (diff !== 0) {
      return diff;
    }
    return a.pid - b.pid;
  });
  return copy;
}

// ProcessUsage is one command averaged over a window, as the server returns it.
export interface ProcessUsage {
  command: string;
  cpu_avg: number;
  cpu_max: number;
  mem_avg: number;
  mem_max: number;
  samples: number;
}

// 'last' is the snapshot from the host's last push; the rest are windows the
// server averages over, and match the ranges the charts already offer.
export const USAGE_WINDOWS = ['last', '1h', '12h', '24h', '7d'] as const;

export type UsageWindow = (typeof USAGE_WINDOWS)[number];

// sortUsage orders a window by one dimension, descending, without mutating the
// caller's array. Ties break on command so rows do not shuffle between polls.
export function sortUsage(rows: ProcessUsage[], by: ProcessSort): ProcessUsage[] {
  const copy = [...rows];
  copy.sort((a, b) => {
    let diff = 0;
    if (by === 'cpu') {
      diff = b.cpu_avg - a.cpu_avg;
    } else {
      diff = b.mem_avg - a.mem_avg;
    }
    if (diff !== 0) {
      return diff;
    }
    return a.command.localeCompare(b.command);
  });
  return copy;
}
