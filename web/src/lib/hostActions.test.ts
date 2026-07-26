import { describe, expect, it } from 'vitest';
import { hostRowActions } from './hostActions';
import type { Host, UpdateState } from '../api';

const host = (status: Host['status'], update_state: UpdateState): Host =>
  ({ id: 'h', name: 'h', status, update_state } as Host);

describe('hostRowActions', () => {
  // The whole point: a machine that is fine offers nothing to click.
  it('offers nothing for an online host on the published build', () => {
    expect(hostRowActions(host('online', 'up_to_date'))).toEqual({ install: false, update: false });
  });

  it('offers update for an online host that is behind', () => {
    expect(hostRowActions(host('online', 'outdated'))).toEqual({ install: true, update: true });
  });

  // update-now refuses a host whose policy is off, so a button would 409.
  it('offers no update when auto-update is off', () => {
    expect(hostRowActions(host('online', 'disabled')).update).toBe(false);
  });

  it('offers update for a host the server cannot place', () => {
    expect(hostRowActions(host('online', 'unknown')).update).toBe(true);
  });

  // Nothing to push a command to, so installing is the only move.
  it('offers install but not update for a host that is not online', () => {
    expect(hostRowActions(host('offline', 'outdated'))).toEqual({ install: true, update: false });
    expect(hostRowActions(host('never', 'unknown'))).toEqual({ install: true, update: false });
  });

  it('offers install for a stalled host', () => {
    expect(hostRowActions(host('online', 'stalled'))).toEqual({ install: true, update: true });
  });
});
