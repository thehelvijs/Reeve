import type { CollectionRef, Tool } from '../api';
import { endpointString } from '../api';
import Modal from './Modal';
import EntityIcon from './EntityIcon';
import StatusPill from './StatusPill';

// CollectionInfoModal shows a collection and the services in it — the
// collection-grouping counterpart to HostInfoModal on the portal.
export default function CollectionInfoModal({
  collection,
  tools,
  onClose,
  onOpenTool,
}: {
  collection: CollectionRef & { description?: string };
  tools: Tool[];
  onClose: () => void;
  onOpenTool: (id: string) => void;
}) {
  return (
    <Modal title="Collection" onClose={onClose}>
      <div className="mt-4 flex items-center gap-3">
        <EntityIcon url={collection.icon_url} name={collection.name} size={40} />
        <div className="min-w-0">
          <p className="truncate text-base font-medium text-content">{collection.name}</p>
          {collection.description && <p className="text-xs text-muted">{collection.description}</p>}
        </div>
      </div>

      <p className="mt-5 text-xs font-medium uppercase tracking-wide text-muted">Services ({tools.length})</p>
      <div className="mt-2 space-y-2">
        {tools.length === 0 && <p className="text-sm text-muted">No services in this collection.</p>}
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
    </Modal>
  );
}
