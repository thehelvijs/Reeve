import type { Host, Tool } from '../api';

export interface HostGroup {
  host: Host | null;
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
