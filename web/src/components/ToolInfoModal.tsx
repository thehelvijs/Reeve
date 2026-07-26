import type { Tool } from '../api';
import { endpointString } from '../api';
import Modal from './Modal';
import EntityIcon from './EntityIcon';
import StatusPill from './StatusPill';
import { Button, Pill } from './ui';

// ToolInfoModal shows a service's details in a popup — opened from the portal
// list, map, graph, and host modal.
export default function ToolInfoModal({
  tool,
  hostName,
  canOpen,
  onOpen,
  onClose,
}: {
  tool: Tool;
  hostName?: string;
  canOpen: boolean;
  onOpen: () => void;
  onClose: () => void;
}) {
  const submitAction = () => {
    if (canOpen) {
      return onOpen;
    }
    return undefined;
  };

  return (
    <Modal title="Service" onClose={onClose} onSubmit={submitAction()}>
      {tool.thumbnail_url && (
        <img src={tool.thumbnail_url} alt="" className="mt-4 h-32 w-full rounded-card border border-hairline object-cover" />
      )}
      <div className="mt-4 flex items-center gap-3">
        <EntityIcon url={tool.icon_url} name={tool.name} size={40} />
        <div className="min-w-0 flex-1">
          <p className="truncate text-base font-medium text-content">{tool.name}</p>
          <p className="truncate font-mono text-xs text-muted">{endpointString(tool) || '—'}</p>
        </div>
        <StatusPill status={tool.status} />
      </div>

      {tool.description && <p className="mt-4 text-sm text-muted">{tool.description}</p>}

      <div className="mt-4 flex flex-wrap gap-1.5">
        {tool.collections.map((c) => (
          <Pill key={c.id}>{c.name}</Pill>
        ))}
        {hostName && <Pill>host: {hostName}</Pill>}
        {tool.tags.map((t) => (
          <Pill key={t}>{t}</Pill>
        ))}
      </div>

      {canOpen && (
        <div className="mt-6 flex justify-end">
          <Button onClick={onOpen}>Open service</Button>
        </div>
      )}
    </Modal>
  );
}
