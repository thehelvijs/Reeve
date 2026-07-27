import type { Host, ToolStatus } from '../api';

export type Tone = 'up' | 'down' | 'warn' | 'muted';

// TOOL_LABEL is the operator-facing name of every tool state, shared by the pills,
// the dashboard tiles and the filter tabs so the three cannot disagree.
//
// "unreachable" is deliberately not "down": the service may still run on a host
// that stopped reporting, and calling that down claims knowledge the server does
// not have. "not monitored" is the honest reading of `unknown`, which means no
// agent source is set rather than something being wrong.
export const TOOL_LABEL: Record<ToolStatus, string> = {
  up: 'up',
  down: 'down',
  agent_offline: 'unreachable',
  unknown: 'not monitored',
};

export const TOOL_TONE: Record<ToolStatus, Tone> = {
  up: 'up',
  down: 'down',
  agent_offline: 'warn',
  unknown: 'muted',
};

export function hostTone(status: Host['status']): Tone {
  if (status === 'online') {
    return 'up';
  }
  if (status === 'offline') {
    return 'down';
  }
  return 'muted';
}

// A unit or container reports its own state word, and there are more of them
// than either daemon documents in one place. Anything unrecognised reads as
// muted rather than as a guess: "dead" is systemd's word for a unit that exited
// cleanly, so painting an unknown word red would cry wolf on a healthy machine.
const UNIT_TONE: Record<string, Tone> = {
  active: 'up',
  running: 'up',
  activating: 'warn',
  deactivating: 'warn',
  restarting: 'warn',
  paused: 'warn',
  reloading: 'warn',
  failed: 'down',
  error: 'down',
  inactive: 'muted',
  exited: 'muted',
  created: 'muted',
  dead: 'muted',
  removing: 'muted',
};

export function unitStateTone(state: string): Tone {
  return UNIT_TONE[state.toLowerCase()] ?? 'muted';
}

// ALERT_LABEL is the operator-facing name of every alert an event can carry.
// The raw type is a server key: `mem_high` is not a phrase, and the UI used to
// print it with one underscore swapped for a space, which left the rest in.
const ALERT_LABEL: Record<string, string> = {
  agent_offline: 'host unreachable',
  down: 'service down',
  log_error: 'error in logs',
  cpu_high: 'CPU high',
  mem_high: 'memory high',
  disk_high: 'disk high',
  temp_high: 'temperature high',
  load_high: 'load high',
  net_high: 'network high',
};

export function alertLabel(type: string): string {
  return ALERT_LABEL[type] ?? type.replaceAll('_', ' ');
}

const DOT: Record<Tone, string> = {
  up: 'bg-up-solid',
  down: 'bg-down-solid',
  warn: 'bg-warn-solid',
  muted: 'bg-idle-solid',
};

// dotClass returns the solid fill for a status dot, which is saturated rather than text-safe.
export function dotClass(tone: Tone): string {
  return DOT[tone];
}
