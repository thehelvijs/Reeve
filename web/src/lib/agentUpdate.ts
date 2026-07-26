import type { AutoUpdatePolicy, UpdateState } from '../api';

export const UPDATE_LABEL: Record<UpdateState, string> = {
  up_to_date: 'up to date',
  outdated: 'outdated',
  updating: 'updating',
  stalled: 'update stalled',
  disabled: 'updates off',
  unknown: 'version unknown',
};

export const UPDATE_TONE: Record<UpdateState, 'muted' | 'up' | 'down' | 'warn'> = {
  up_to_date: 'up',
  outdated: 'warn',
  updating: 'muted',
  stalled: 'down',
  disabled: 'muted',
  unknown: 'muted',
};

export const POLICY_LABEL: Record<AutoUpdatePolicy, string> = {
  default: 'Follow the fleet default',
  on: 'Always update',
  off: 'Never update',
};

// An up-to-date host reads as plain text; a pill on every row is noise.
export function showsVersionPill(state: UpdateState): boolean {
  return state !== 'up_to_date';
}
