import { describe, expect, it } from 'vitest';
import { dotClass, hostTone, TOOL_LABEL, TOOL_TONE, type Tone } from './statusTone';
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
