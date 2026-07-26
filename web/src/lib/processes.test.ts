import { describe, expect, it } from 'vitest';
import { sortProcesses, type ProcessSample } from './processes';

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
