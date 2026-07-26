import { describe, expect, it } from 'vitest';
import { toolUpdateBody } from './tools';
import type { Tool } from '../api';

const tool: Tool = {
  id: 't1',
  name: 'Grafana',
  slug: 'grafana',
  description: 'dashboards',
  collections: [
    { id: 'c1', name: 'ops', icon_url: '' },
    { id: 'c2', name: 'prod', icon_url: '' },
  ],
  tags: ['metrics'],
  scheme: 'https',
  address: '10.0.0.5',
  port: 3000,
  url: '',
  physical_location: 'rack 2',
  host_id: 'h1',
  source_type: 'manual',
  source_ref: 'grafana.service',
  status: 'up',
  visibility: 'public',
  creator_id: 'u1',
  icon_url: '',
  thumbnail_url: '',
  can_edit: true,
  log_alert_enabled: true,
  created_at: '2026-07-26T00:00:00Z',
} as Tool;

describe('toolUpdateBody', () => {
  // PATCH replaces every field it is given, so a one-field change that drops
  // any other field silently wipes it on the server.
  it('carries every field the tool already has', () => {
    expect(toolUpdateBody(tool)).toEqual({
      name: 'Grafana',
      slug: 'grafana',
      description: 'dashboards',
      collection_ids: ['c1', 'c2'],
      tags: ['metrics'],
      scheme: 'https',
      address: '10.0.0.5',
      port: 3000,
      url: '',
      physical_location: 'rack 2',
      host_id: 'h1',
      source_type: 'manual',
      source_ref: 'grafana.service',
      visibility: 'public',
      log_alert_enabled: true,
    });
  });

  it('applies the change over the tool', () => {
    expect(toolUpdateBody(tool, { visibility: 'restricted' }).visibility).toBe('restricted');
  });

  it('sends absent optionals as empty rather than undefined', () => {
    const bare = { ...tool, port: undefined, url: undefined, physical_location: undefined, host_id: undefined };
    const body = toolUpdateBody(bare);
    expect(body.port).toBe(0);
    expect(body.url).toBe('');
    expect(body.physical_location).toBe('');
    expect(body.host_id).toBe('');
  });
});
