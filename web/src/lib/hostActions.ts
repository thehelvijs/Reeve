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
  // The server's own row has no agent to install or update.
  if (host.is_server) {
    return { install: false, update: false };
  }
  const healthy = host.status === 'online' && host.update_state === 'up_to_date';
  return {
    // The fallback for a machine that is not reporting, or reporting a build
    // the server cannot place. Pushing the installer fixes both.
    install: !healthy,
    // A manual push for anything that will not get there on its own, which is
    // the whole point of the button: auto-update off means "not on the paced
    // rollout", not "never". The one exception is the machine's own veto — the
    // agent ignores the ack, so a button there would lie.
    update: host.status === 'online' && host.update_state !== 'up_to_date' && !host.auto_update_vetoed,
  };
}
