import type { Host, Tool } from '../api';
import Modal from './Modal';
import EntityIcon from './EntityIcon';
import ToolRows from './ToolRows';
import { Pill } from './ui';

function statusTone(status: Host['status']): 'up' | 'down' | 'muted' {
  if (status === 'online') {
    return 'up';
  }
  if (status === 'offline') {
    return 'down';
  }
  return 'muted';
}

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
            {host.ip_address ? ` · ${host.ip_address}` : ''}
            {host.latitude != null && host.longitude != null
              ? ` · ${host.latitude.toFixed(2)}, ${host.longitude.toFixed(2)}`
              : ''}
          </p>
        </div>
        <Pill tone={statusTone(host.status)}>{host.status}</Pill>
      </div>

      <ToolRows tools={tools} empty="No services on this host." onOpenTool={onOpenTool} />
    </Modal>
  );
}
