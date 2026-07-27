import { describe, expect, it } from 'vitest';
import { hostMatchesFilter, HOST_FILTERS } from './hostFilter';
import type { Host, UpdateState } from '../api';

const host = (status: Host['status'], update_state: UpdateState = 'up_to_date'): Host =>
  ({ id: 'h', name: 'h', status, update_state } as Host);

describe('hostMatchesFilter', () => {
  it('keeps every host under All', () => {
    expect(hostMatchesFilter(host('never'), 'all')).toBe(true);
    expect(hostMatchesFilter(host('online'), 'all')).toBe(true);
  });

  // "Not reporting" has to include a host that never checked in, or a count of silent machines misses one.
  it('counts never-seen hosts as not reporting', () => {
    expect(hostMatchesFilter(host('never'), 'quiet')).toBe(true);
    expect(hostMatchesFilter(host('offline'), 'quiet')).toBe(true);
    expect(hostMatchesFilter(host('online'), 'quiet')).toBe(false);
  });

  it('separates online from not reporting', () => {
    expect(hostMatchesFilter(host('online'), 'online')).toBe(true);
    expect(hostMatchesFilter(host('offline'), 'online')).toBe(false);
  });

  // "Agent behind" means a build the rollout still owes, not updates switched off or an unplaceable version.
  it('counts outdated and stalled builds as behind', () => {
    expect(hostMatchesFilter(host('online', 'outdated'), 'stale')).toBe(true);
    expect(hostMatchesFilter(host('online', 'stalled'), 'stale')).toBe(true);
    expect(hostMatchesFilter(host('online', 'up_to_date'), 'stale')).toBe(false);
    expect(hostMatchesFilter(host('online', 'disabled'), 'stale')).toBe(false);
    expect(hostMatchesFilter(host('online', 'unknown'), 'stale')).toBe(false);
  });

  // Every tab has to be reachable, or a count renders with nothing behind it.
  it('exposes a predicate for every tab', () => {
    for (const f of HOST_FILTERS) {
      expect(typeof hostMatchesFilter(host('online'), f.key)).toBe('boolean');
    }
  });
});
