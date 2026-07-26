import type { Tool } from '../api';
import { endpointString } from '../api';
import EntityIcon from './EntityIcon';
import StatusPill from './StatusPill';

// The service list shared by the host and collection modals: a counted heading
// and one clickable row per service, so both read identically.
export default function ToolRows({
  tools,
  empty,
  onOpenTool,
}: {
  tools: Tool[];
  empty: string;
  onOpenTool: (id: string) => void;
}) {
  return (
    <>
      <p className="mt-5 text-xs font-medium uppercase tracking-wide text-muted">
        Services ({tools.length})
      </p>
      <div className="mt-2 space-y-2">
        {tools.length === 0 && <p className="text-sm text-muted">{empty}</p>}
        {tools.map((t) => (
          <button
            key={t.id}
            type="button"
            onClick={() => onOpenTool(t.id)}
            className="flex w-full items-center gap-3 rounded-button border border-hairline bg-surface-2/40 px-3 py-2 text-left transition-colors hover:bg-surface-2"
          >
            <EntityIcon url={t.icon_url} name={t.name} size={28} />
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm text-content">{t.name}</p>
              <p className="truncate font-mono text-xs text-muted">{endpointString(t) || '—'}</p>
            </div>
            <StatusPill status={t.status} />
          </button>
        ))}
      </div>
    </>
  );
}
