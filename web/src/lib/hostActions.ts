import type { Host } from '../api';

export interface RowActions {
  install: boolean;
  update: boolean;
}

// What a host row offers. A machine that is online and running the published
// build needs nothing done to it, so it gets no buttons — the row is for
// reading, and every button on it is one more thing to misclick.
//
// Removing the agent is deliberately not here: it belongs with deleting the
// host, on the host's own page, not one click away in a list.
export function hostRowActions(host: Host): RowActions {
  const healthy = host.status === 'online' && host.update_state === 'up_to_date';
  return {
    // The fallback for a machine that is not reporting, or reporting a build
    // the server cannot place. Pushing the installer fixes both.
    install: !healthy,
    // Only where the server accepts the override. A host whose auto-update is
    // off is refused by design, and the fix there is the policy on its page,
    // so a button here would be a 409 waiting to happen.
    update:
      host.status === 'online' && host.update_state !== 'up_to_date' && host.update_state !== 'disabled',
  };
}
