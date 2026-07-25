import type { CollectionRef, Host, Tool } from '../api';

export interface HostGroup {
  host: Host | null;
  tools: Tool[];
}

export interface CollectionGroup {
  collection: CollectionRef | null;
  tools: Tool[];
}

export function groupToolsByHost(tools: Tool[], hosts: Host[]): HostGroup[] {
  const byHost = new Map<string, Tool[]>();
  const unassigned: Tool[] = [];
  for (const t of tools) {
    if (t.host_id) {
      const list = byHost.get(t.host_id) ?? [];
      list.push(t);
      byHost.set(t.host_id, list);
    } else {
      unassigned.push(t);
    }
  }
  const groups: HostGroup[] = [];
  for (const h of hosts) {
    const list = byHost.get(h.id);
    if (list && list.length > 0) {
      groups.push({ host: h, tools: list });
    }
  }
  if (unassigned.length > 0) {
    groups.push({ host: null, tools: unassigned });
  }
  return groups;
}

// A tool in two visible collections is listed under both, deliberately.
export function groupToolsByCollection(tools: Tool[]): CollectionGroup[] {
  const byCollection = new Map<string, { collection: CollectionRef; tools: Tool[] }>();
  const ungrouped: Tool[] = [];
  for (const t of tools) {
    if (t.collections.length === 0) {
      ungrouped.push(t);
      continue;
    }
    for (const c of t.collections) {
      const entry = byCollection.get(c.id);
      if (entry) {
        entry.tools.push(t);
      } else {
        byCollection.set(c.id, { collection: c, tools: [t] });
      }
    }
  }
  const groups: CollectionGroup[] = Array.from(byCollection.values()).sort((a, b) =>
    a.collection.name.localeCompare(b.collection.name),
  );
  if (ungrouped.length > 0) {
    groups.push({ collection: null, tools: ungrouped });
  }
  return groups;
}
