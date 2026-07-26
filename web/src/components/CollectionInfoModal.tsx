import type { CollectionRef, Tool } from '../api';
import Modal from './Modal';
import EntityIcon from './EntityIcon';
import ToolRows from './ToolRows';

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

      <ToolRows tools={tools} empty="No services in this collection." onOpenTool={onOpenTool} />
    </Modal>
  );
}
