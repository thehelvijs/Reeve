import { describe, expect, it } from 'vitest';
import { hostRowActions } from './hostActions';
import type { Host, UpdateState } from '../api';

const host = (status: Host['status'], update_state: UpdateState, vetoed = false): Host =>
  ({ id: 'h', name: 'h', status, update_state, auto_update_vetoed: vetoed } as Host);

describe('hostRowActions', () => {
  // The whole point: a machine that is fine offers nothing to click.
  it('offers nothing for an online host on the published build', () => {
    expect(hostRowActions(host('online', 'up_to_date'))).toEqual({ install: false, update: false });
  });

  it('offers update for an online host that is behind', () => {
    expect(hostRowActions(host('online', 'outdated'))).toEqual({ install: true, update: true });
  });

  // Auto-update off means "not on the paced rollout", not "never" — a manual
  // push is exactly what the button is for on a host nothing else will reach.
  it('offers update when auto-update is off', () => {
    expect(hostRowActions(host('online', 'disabled')).update).toBe(true);
  });

  // The machine's own veto is the one case the server cannot act on, because
  // the agent ignores the ack. A button there would lie.
  it('offers no update when the machine itself vetoed', () => {
    expect(hostRowActions(host('online', 'disabled', true)).update).toBe(false);
    expect(hostRowActions(host('online', 'outdated', true)).update).toBe(false);
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
