import type { Tool } from '../api';

// toolUpdateBody renders a tool back into the shape PATCH /api/tools/{id}
// takes. That endpoint replaces every field it is given, so changing one means
// resending the rest exactly as the row already has them.
export function toolUpdateBody(t: Tool, changes: Partial<Tool> = {}) {
  const next = { ...t, ...changes };
  return {
    name: next.name,
    slug: next.slug,
    description: next.description,
    collection_ids: next.collections.map((c) => c.id),
    tags: next.tags,
    scheme: next.scheme,
    address: next.address,
    port: next.port ?? 0,
    url: next.url ?? '',
    physical_location: next.physical_location ?? '',
    host_id: next.host_id ?? '',
    source_type: next.source_type,
    source_ref: next.source_ref,
    visibility: next.visibility,
    log_alert_enabled: next.log_alert_enabled,
  };
}
