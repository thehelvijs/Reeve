import { describe, expect, it } from 'vitest';
import { alertLabel, dotClass, hostTone, TOOL_LABEL, TOOL_TONE, unitStateTone, type Tone } from './statusTone';
import type { ToolStatus } from '../api';

const ALL: ToolStatus[] = ['up', 'down', 'agent_offline', 'unknown'];

describe('tool status vocabulary', () => {
  // The bug: agent_offline read exactly like down, so "3 down" could mean three healthy services on one quiet host.
  it('separates unreachable from down', () => {
    expect(TOOL_LABEL.agent_offline).not.toBe(TOOL_LABEL.down);
    expect(TOOL_TONE.agent_offline).not.toBe(TOOL_TONE.down);
  });

  // "unknown" is the server's name for "no agent source set", and showing it raw is what made the state unreadable.
  it('names unknown as not monitored', () => {
    expect(TOOL_LABEL.unknown).toBe('not monitored');
  });

  it('gives every status a label and a tone', () => {
    for (const s of ALL) {
      expect(TOOL_LABEL[s]).toBeTruthy();
      expect(TOOL_TONE[s]).toBeTruthy();
    }
  });
});

describe('hostTone', () => {
  it('maps reporting, silent and never-seen apart', () => {
    expect(hostTone('online')).toBe('up');
    expect(hostTone('offline')).toBe('down');
    expect(hostTone('never')).toBe('muted');
  });
});

describe('dotClass', () => {
  // A dot with no fill class is an invisible dot, which reads as "no status".
  it('returns a distinct fill for every tone', () => {
    const tones: Tone[] = ['up', 'down', 'warn', 'muted'];
    for (const t of tones) {
      expect(dotClass(t)).toMatch(/^bg-/);
    }
    expect(new Set(tones.map(dotClass)).size).toBe(tones.length);
  });
});

describe('unitStateTone', () => {
  it('separates running from failed from stopped', () => {
    expect(unitStateTone('active')).toBe('up');
    expect(unitStateTone('running')).toBe('up');
    expect(unitStateTone('failed')).toBe('down');
    expect(unitStateTone('inactive')).toBe('muted');
    expect(unitStateTone('exited')).toBe('muted');
  });

  it('reads a state mid-transition as a warning, not as healthy', () => {
    expect(unitStateTone('activating')).toBe('warn');
    expect(unitStateTone('restarting')).toBe('warn');
  });

  // A word neither daemon documents must not be painted red: "dead" is how
  // systemd describes a unit that exited cleanly.
  it('falls back to muted for a word it does not know', () => {
    expect(unitStateTone('whatever-systemd-adds-next')).toBe('muted');
    expect(unitStateTone('ACTIVE')).toBe('up');
  });
});

describe('alertLabel', () => {
  // The bug: the raw type went to screen with one underscore swapped for a
  // space, so mem_high read as "mem high" and log_error as "log error".
  it('names every alert the server fires as a phrase', () => {
    for (const metric of ['cpu', 'mem', 'disk', 'temp', 'load', 'net']) {
      expect(alertLabel(`${metric}_high`)).not.toContain('_');
    }
    expect(alertLabel('agent_offline')).toBe('host unreachable');
    expect(alertLabel('down')).toBe('service down');
  });

  it('leaves no underscore in a type it has never seen', () => {
    expect(alertLabel('some_new_alert')).toBe('some new alert');
  });
});
