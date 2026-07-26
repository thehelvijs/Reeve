import { describe, expect, it } from 'vitest';
import { endpointString, goURL, type Tool } from './api';

// A Tool with only the fields these functions read; the rest never affects them.
function tool(fields: Partial<Tool>): Tool {
  return {
    id: 't1',
    name: 'Thing',
    slug: 'thing',
    description: '',
    collections: [],
    tags: [],
    scheme: '',
    address: '',
    source_type: 'manual',
    source_ref: '',
    status: 'unknown',
    visibility: 'public',
    creator_id: 'u1',
    can_edit: true,
    log_alert_enabled: false,
    icon_url: '',
    thumbnail_url: '',
    ...fields,
  } as Tool;
}

describe('endpointString', () => {
  it('prefers an explicit URL over the address parts', () => {
    expect(endpointString(tool({ url: 'https://grafana.example', address: 'ignored', port: 3000 }))).toBe(
      'https://grafana.example',
    );
  });

  it('assembles scheme, address and port only when each is present', () => {
    expect(endpointString(tool({ address: '192.168.1.20' }))).toBe('192.168.1.20');
    expect(endpointString(tool({ address: '192.168.1.20', port: 8080 }))).toBe('192.168.1.20:8080');
    expect(endpointString(tool({ scheme: 'http', address: '192.168.1.20', port: 8080 }))).toBe(
      'http://192.168.1.20:8080',
    );
    expect(endpointString(tool({ scheme: 'http', address: '192.168.1.20' }))).toBe('http://192.168.1.20');
  });

  // A blank address is how a tool opts into following its host, and only the
  // server knows where that host is now: a placeholder beats a wrong answer.
  it('says it follows the host rather than guessing an address', () => {
    expect(endpointString(tool({ host_id: 'h1' }))).toBe('follows this host');
    expect(endpointString(tool({}))).toBe('');
  });

  it('treats port 0 as no port', () => {
    expect(endpointString(tool({ address: 'box', port: 0 }))).toBe('box');
  });
});

describe('goURL', () => {
  it('is the slug path the server resolves at click time', () => {
    expect(goURL({ slug: 'paperless-ngx' })).toBe('/go/paperless-ngx');
  });
});
