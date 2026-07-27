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
