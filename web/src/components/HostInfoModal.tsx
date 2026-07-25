import type { Host, Tool } from '../api';
import { endpointString } from '../api';
import Modal from './Modal';
import EntityIcon from './EntityIcon';
import StatusPill from './StatusPill';
import { Pill } from './ui';

// HostInfoModal shows a host's details and its services in a popup — used from
// the portal list, map, and graph so a click anywhere opens the same view.
export default function HostInfoModal({
  host,
  tools,
  onClose,
  onOpenTool,
}: {
  host: Host;
  tools: Tool[];
  onClose: () => void;
  onOpenTool: (id: string) => void;
}) {
  const tone = host.status === 'online' ? 'up' : host.status === 'offline' ? 'down' : 'muted';
  return (
    <Modal title="Host" onClose={onClose}>
      {host.thumbnail_url && (
        <img src={host.thumbnail_url} alt="" className="mt-4 h-32 w-full rounded-card border border-hairline object-cover" />
      )}
      <div className="mt-4 flex items-center gap-3">
        <EntityIcon url={host.icon_url} name={host.name} size={40} />
        <div className="min-w-0">
          <p className="truncate text-base font-medium text-content">{host.name}</p>
          <p className="text-xs text-muted">
            {host.os || 'host'}
            {host.latitude != null && host.longitude != null
              ? ` · ${host.latitude.toFixed(2)}, ${host.longitude.toFixed(2)}`
              : ''}
          </p>
        </div>
        <Pill tone={tone}>{host.status}</Pill>
      </div>

      <p className="mt-5 text-xs font-medium uppercase tracking-wide text-muted">Services ({tools.length})</p>
      <div className="mt-2 space-y-2">
        {tools.length === 0 && <p className="text-sm text-muted">No services on this host.</p>}
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
