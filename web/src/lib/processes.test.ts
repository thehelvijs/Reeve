import { describe, expect, it } from 'vitest';
import { sortProcesses, sortUsage, type ProcessSample, type ProcessUsage } from './processes';

const p = (pid: number, cpu: number, mem: number): ProcessSample => ({
  pid,
  user: 'root',
  command: `cmd-${pid}`,
  cpu_pct: cpu,
  mem_rss: mem,
});

describe('sortProcesses', () => {
  it('orders by cpu descending', () => {
    const got = sortProcesses([p(1, 5, 100), p(2, 90, 1), p(3, 40, 50)], 'cpu');
    expect(got.map((x) => x.pid)).toEqual([2, 3, 1]);
  });

  it('orders by memory descending', () => {
    const got = sortProcesses([p(1, 5, 100), p(2, 90, 1), p(3, 40, 50)], 'mem');
    expect(got.map((x) => x.pid)).toEqual([1, 3, 2]);
  });

  it('breaks ties on pid so rows do not shuffle between polls', () => {
    const got = sortProcesses([p(9, 0, 0), p(2, 0, 0), p(5, 0, 0)], 'cpu');
    expect(got.map((x) => x.pid)).toEqual([2, 5, 9]);
  });

  it('does not mutate its input', () => {
    const input = [p(1, 5, 100), p(2, 90, 1)];
    sortProcesses(input, 'cpu');
    expect(input.map((x) => x.pid)).toEqual([1, 2]);
  });

  it('handles an empty snapshot', () => {
    expect(sortProcesses([], 'cpu')).toEqual([]);
  });
});

const u = (command: string, cpu: number, mem: number): ProcessUsage => ({
  command,
  cpu_avg: cpu,
  cpu_max: cpu * 2,
  mem_avg: mem,
  mem_max: mem * 2,
  samples: 10,
});

describe('sortUsage', () => {
  it('orders by average CPU, descending', () => {
    const got = sortUsage([u('a', 5, 100), u('b', 90, 1), u('c', 40, 50)], 'cpu');
    expect(got.map((r) => r.command)).toEqual(['b', 'c', 'a']);
  });

  it('orders by average memory, descending', () => {
    const got = sortUsage([u('a', 5, 100), u('b', 90, 1), u('c', 40, 50)], 'mem');
    expect(got.map((r) => r.command)).toEqual(['a', 'c', 'b']);
  });

  // A row order that changes between polls moves the row under the cursor.
  it('breaks ties on command so the order is stable', () => {
    const got = sortUsage([u('zeta', 1, 1), u('alpha', 1, 1)], 'cpu');
    expect(got.map((r) => r.command)).toEqual(['alpha', 'zeta']);
  });

  it('does not mutate the input', () => {
    const rows = [u('a', 1, 1), u('b', 9, 9)];
    sortUsage(rows, 'cpu');
    expect(rows.map((r) => r.command)).toEqual(['a', 'b']);
  });
});
