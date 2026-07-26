import { describe, expect, it } from 'vitest';
import type { CollectionRef, Host, Tool } from '../api';
import { groupToolsByCollection, groupToolsByHost } from './group';

function tool(id: string, fields: Partial<Tool> = {}): Tool {
  return { id, name: id, slug: id, collections: [], ...fields } as Tool;
}

function host(id: string): Host {
  return { id, name: id } as Host;
}

function collection(id: string, name: string): CollectionRef {
  return { id, name } as CollectionRef;
}

describe('groupToolsByHost', () => {
  it('follows the order of the hosts list, not the tools list', () => {
    const groups = groupToolsByHost(
      [tool('a', { host_id: 'h2' }), tool('b', { host_id: 'h1' })],
      [host('h1'), host('h2')],
    );
    expect(groups.map((g) => g.host?.id)).toEqual(['h1', 'h2']);
  });

  it('puts host-less tools in a trailing null group', () => {
    const groups = groupToolsByHost([tool('a'), tool('b', { host_id: 'h1' })], [host('h1')]);
    expect(groups).toHaveLength(2);
    expect(groups[1].host).toBeNull();
    expect(groups[1].tools.map((t) => t.id)).toEqual(['a']);
  });

  it('omits a host with no tools instead of showing an empty group', () => {
    const groups = groupToolsByHost([tool('a', { host_id: 'h1' })], [host('h1'), host('h2')]);
    expect(groups.map((g) => g.host?.id)).toEqual(['h1']);
  });

  // A tool pointing at a host that is not in the list — deleted, or not visible
  // to this viewer — must not vanish from the page entirely.
  it('drops a tool whose host is absent from the hosts list', () => {
    const groups = groupToolsByHost([tool('a', { host_id: 'gone' })], []);
    expect(groups).toEqual([]);
  });

  it('returns nothing for no tools', () => {
    expect(groupToolsByHost([], [host('h1')])).toEqual([]);
  });
});

describe('groupToolsByCollection', () => {
  it('sorts collections by name and trails the ungrouped', () => {
    const groups = groupToolsByCollection([
      tool('a', { collections: [collection('c2', 'Zebra')] }),
      tool('b', { collections: [collection('c1', 'Alpha')] }),
      tool('c'),
    ]);
    expect(groups.map((g) => g.collection?.name)).toEqual(['Alpha', 'Zebra', undefined]);
    expect(groups[2].collection).toBeNull();
  });

  it('lists a tool in every collection it belongs to', () => {
    const groups = groupToolsByCollection([
      tool('a', { collections: [collection('c1', 'Alpha'), collection('c2', 'Beta')] }),
    ]);
    expect(groups).toHaveLength(2);
    expect(groups.every((g) => g.tools.map((t) => t.id).includes('a'))).toBe(true);
  });

  it('merges tools that share a collection into one group', () => {
    const c = collection('c1', 'Alpha');
    const groups = groupToolsByCollection([tool('a', { collections: [c] }), tool('b', { collections: [c] })]);
    expect(groups).toHaveLength(1);
    expect(groups[0].tools.map((t) => t.id)).toEqual(['a', 'b']);
  });

  it('returns nothing for no tools', () => {
    expect(groupToolsByCollection([])).toEqual([]);
  });
});
