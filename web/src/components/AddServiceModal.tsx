import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, type Host, type HostInventory, type InventoryItem } from '../api';
import { Button, Pill } from './ui';
import Modal from './Modal';
import EntityIcon from './EntityIcon';
import Chevron from './Chevron';

// AddServiceModal is the "what do you want to monitor" step: pick a host, pick
// something the agent already found on it, and the form opens prefilled. The
// host page has always been able to do this for one host; this reaches every
// host without knowing which one first.
//
// Manual entry stays one click away throughout, because a service on a machine
// with no agent is a normal thing to add and there is nothing to discover.
export default function AddServiceModal({ onClose }: { onClose: () => void }) {
  const navigate = useNavigate();
  const [hosts, setHosts] = useState<Host[] | null>(null);
  const [picked, setPicked] = useState<Host | null>(null);

  useEffect(() => {
    api
      .get<Host[]>('/api/hosts')
      .then((h) => setHosts(h ?? []))
      .catch(() => setHosts([]));
  }, []);

  const manual = (host?: Host) => {
    const q = new URLSearchParams();
    if (host) {
      q.set('host_id', host.id);
    }
    const query = q.toString();
    if (query) {
      navigate(`/services/new?${query}`);
    } else {
      navigate('/services/new');
    }
  };

  if (picked) {
    return <HostItems host={picked} onBack={() => setPicked(null)} onManual={() => manual(picked)} onClose={onClose} />;
  }

  return (
    <Modal title="Add for monitoring" onClose={onClose} onSubmit={() => manual()}>
      <p className="mt-2 text-sm text-muted">
        Pick the machine it runs on and choose from what its agent already found, or enter it by hand.
      </p>

      {hosts === null && <p className="mt-4 text-sm text-muted">Loading hosts…</p>}
      {hosts !== null && hosts.length === 0 && (
        <p className="mt-4 text-sm text-muted">
          No hosts yet. You can still add a service by hand and point it at any address.
        </p>
      )}
      {hosts !== null && hosts.length > 0 && (
        <div className="mt-4 max-h-72 divide-y divide-hairline overflow-y-auto rounded-card border border-hairline">
          {hosts.map((h) => (
            <button
              key={h.id}
              type="button"
              onClick={() => setPicked(h)}
              className="flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors hover:bg-surface-2"
            >
              <EntityIcon url={h.icon_url} name={h.name} size={28} />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm text-content">{h.name}</p>
                <p className="truncate text-xs text-muted">{h.os}</p>
              </div>
              <Pill tone={h.status === 'online' ? 'up' : 'muted'}>{h.status}</Pill>
              <Chevron />
            </button>
          ))}
        </div>
      )}

      <div className="mt-5 flex justify-end gap-2">
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        <Button onClick={() => manual()}>Add manually</Button>
      </div>
    </Modal>
  );
}

// HostItems lists what one host's agent reported, already-linked entries
// included but marked, so it is obvious why something is not offered again.
function HostItems({
  host,
  onBack,
  onManual,
  onClose,
}: {
  host: Host;
  onBack: () => void;
  onManual: () => void;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const [inv, setInv] = useState<HostInventory | null>(null);

  useEffect(() => {
    api
      .get<HostInventory>(`/api/hosts/${host.id}/inventory`)
      .then((i) => setInv(i))
      .catch(() => setInv({ services: [], containers: [], cron_jobs: [] }));
  }, [host.id]);

  const add = (item: InventoryItem) => {
    const q = new URLSearchParams({
      name: item.name,
      host_id: host.id,
      source_type: item.source_type,
      source_ref: item.source_ref,
    });
    navigate(`/services/new?${q}`);
  };

  const groups: { label: string; items: InventoryItem[] }[] = [
    { label: 'Systemd units', items: inv?.services ?? [] },
    { label: 'Containers', items: inv?.containers ?? [] },
  ];
  const total = groups.reduce((n, g) => n + g.items.length, 0);

  return (
    <Modal title={`On ${host.name}`} onClose={onClose} onSubmit={onManual}>
      {inv === null && <p className="mt-4 text-sm text-muted">Loading what the agent found…</p>}
      {inv !== null && total === 0 && (
        <p className="mt-4 text-sm text-muted">
          Its agent has not reported any units or containers — it may not have pushed yet, or there is nothing running
          that Reeve discovers. Add the service by hand instead.
        </p>
      )}
      {inv !== null && total > 0 && (
        <div className="mt-4 max-h-72 space-y-4 overflow-y-auto">
          {groups.map((g) =>
            g.items.length === 0 ? null : (
              <div key={g.label}>
                <p className="text-xs font-medium text-muted">
                  {g.label} <span className="text-muted">({g.items.length})</span>
                </p>
                <div className="mt-1.5 divide-y divide-hairline rounded-card border border-hairline">
                  {g.items.map((it) => (
                    <div key={it.source_ref} className="flex items-center justify-between gap-3 px-3 py-2">
                      <div className="min-w-0">
                        <p className="truncate text-sm text-content">{it.name}</p>
                        <p className="truncate font-mono text-xs text-muted">{it.detail}</p>
                      </div>
                      {it.linked ? (
                        <Pill tone="up">linked</Pill>
                      ) : (
                        <Button variant="secondary" onClick={() => add(it)}>
                          Add
                        </Button>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ),
          )}
        </div>
      )}

      <div className="mt-5 flex justify-end gap-2">
        <Button variant="secondary" onClick={onBack}>
          Back
        </Button>
        <Button onClick={onManual}>Add manually</Button>
      </div>
    </Modal>
  );
}
