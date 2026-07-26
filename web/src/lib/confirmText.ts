// What a per-row control asks before it runs. Starting something that is down
// is not destructive and stays one click; stop and restart take whatever is
// using it down with them, so they ask first.
const NEEDS_CONFIRM = new Set(['stop', 'restart']);

export interface Confirmation {
  title: string;
  body: string;
  confirmLabel: string;
}

export function needsConfirm(verb: string): boolean {
  return NEEDS_CONFIRM.has(verb);
}

// rowConfirmation names the target in the question, because these rows sit in a
// scrolling list where "restart?" alone does not say restart what.
export function rowConfirmation(verb: string, target: string, hostName: string): Confirmation {
  const what = `${target} on ${hostName}`;
  if (verb === 'stop') {
    return {
      title: `Stop ${target}?`,
      body: `${what} stops as soon as its agent picks this up (~15s). Anything depending on it goes down until it is started again.`,
      confirmLabel: 'Stop',
    };
  }
  return {
    title: `Restart ${target}?`,
    body: `${what} restarts as soon as its agent picks this up (~15s). It is unavailable while it comes back.`,
    confirmLabel: 'Restart',
  };
}
