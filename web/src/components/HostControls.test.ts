import { describe, expect, it } from 'vitest';
import { unavailableReason } from './HostControls';
import type { Host } from '../api';

const host = (over: Partial<Host>): Host =>
  ({
    id: 'h',
    name: 'box',
    os: 'linux',
    ip_address: '10.0.0.1',
    agent_version: '0.1.0',
    status: 'online',
    icon_url: '',
    thumbnail_url: '',
    auto_update: 'default',
    update_state: 'current',
    control_enabled: true,
    ...over,
  }) as Host;

describe('unavailableReason', () => {
  it('allows controls on an online agent that reported control support', () => {
    expect(unavailableReason(host({}))).toBe('');
  });

  it('blocks an offline host, because nothing would collect the command', () => {
    expect(unavailableReason(host({ status: 'offline' }))).toMatch(/offline/i);
    expect(unavailableReason(host({ status: 'never' }))).toMatch(/offline/i);
  });

  it('blocks a host whose agent never reported control support', () => {
    expect(unavailableReason(host({ control_enabled: false }))).toMatch(/REEVE_ALLOW_CONTROL/);
  });

  it('reports offline first: it is the cause the operator can act on', () => {
    expect(unavailableReason(host({ status: 'offline', control_enabled: false }))).toMatch(/offline/i);
  });
});
