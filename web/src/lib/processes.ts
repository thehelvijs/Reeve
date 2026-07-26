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
