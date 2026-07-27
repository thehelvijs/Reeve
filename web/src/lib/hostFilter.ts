import type { Host } from '../api';

export type HostFilter = 'all' | 'online' | 'quiet' | 'stale';

// The tabs on the Hosts page. "Quiet" folds offline and never-seen together
// because both mean the same thing to an operator: nothing is reporting. "Stale"
// is the actionable one — a build the rollout has not delivered yet.
export const HOST_FILTERS: { key: HostFilter; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'online', label: 'Online' },
  { key: 'quiet', label: 'Not reporting' },
  { key: 'stale', label: 'Agent behind' },
];

export function hostMatchesFilter(host: Host, filter: HostFilter): boolean {
  switch (filter) {
    case 'online':
      return host.status === 'online';
    case 'quiet':
      return host.status !== 'online';
    case 'stale':
      return host.update_state === 'outdated' || host.update_state === 'stalled';
    default:
      return true;
  }
}
