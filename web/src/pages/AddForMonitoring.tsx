import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, type Host, type HostInventory, type InventoryItem } from '../api';
import { Button, Card, Pill, Section, Table, Tabs, Td, Th, Tr } from '../components/ui';
import PageHeader from '../components/PageHeader';
import EmptyState from '../components/EmptyState';
import EntityIcon from '../components/EntityIcon';
import { ListSkeleton } from '../components/Skeleton';
import { matchesQuery } from '../lib/search';

// AddForMonitoring is everything the agents have found, on one searchable page:
// units, containers and cron jobs across every host, each one click from a
// prefilled service form. It replaced a modal that could only show one host at a
// time, which meant knowing which machine a thing was on before looking for it.

type Kind = 'all' | 'systemd' | 'docker' | 'cron';

const KINDS: { key: Kind; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'systemd', label: 'Systemd units' },
  { key: 'docker', label: 'Containers' },
  { key: 'cron', label: 'Cron jobs' },
];

const KIND_LABEL: Record<string, string> = {
  systemd: 'Systemd unit',
  docker: 'Container',
  cron: 'Cron job',
};

interface Found {
  host: Host;
  item: InventoryItem;
}

// AddCell is the one thing a row offers: add it, or say why it is not on offer.
function AddCell({ found, onAdd }: { found: Found; onAdd: () => void }) {
  if (found.item.linked) {
    return <Pill tone="up">linked</Pill>;
  }
  return (
    <Button variant="secondary" onClick={onAdd}>
      Add
    </Button>
  );
}

export default function AddForMonitoring() {
  const navigate = useNavigate();
  const [found, setFound] = useState<Found[] | null>(null);
  const [hostCount, setHostCount] = useState(0);
  const [search, setSearch] = useState('');
  const [kind, setKind] = useState<Kind>('all');

  // One request per host. The fan-out is the client's because there is no
  // fleet-wide inventory endpoint and a LAN fleet is tens of machines, not
  // thousands; a host whose inventory fails to load simply contributes nothing.
  useEffect(() => {
    let live = true;
    api
      .get<Host[]>('/api/hosts')
      .then(async (hosts) => {
        const list = hosts ?? [];
        setHostCount(list.length);
        const per = await Promise.all(
          list.map(async (host) => {
            try {
              const inv = await api.get<HostInventory>(`/api/hosts/${host.id}/inventory`);
              return [...(inv.services ?? []), ...(inv.containers ?? []), ...(inv.cron_jobs ?? [])].map(
                (item) => ({ host, item }),
              );
            } catch {
              return [];
            }
          }),
        );
        if (live) {
          setFound(per.flat());
        }
      })
      .catch(() => {
        if (live) {
          setFound([]);
        }
      });
    return () => {
      live = false;
    };
  }, []);

  const add = (f: Found) => {
    const q = new URLSearchParams({
      name: f.item.name,
      host_id: f.host.id,
      source_type: f.item.source_type,
      source_ref: f.item.source_ref,
    });
    navigate(`/services/new?${q}`);
  };

  const matched = useMemo(() => {
    return (found ?? []).filter((f) =>
      matchesQuery(search, f.item.name, f.item.detail, f.host.name, KIND_LABEL[f.item.source_type]),
    );
  }, [found, search]);
  const shown = matched.filter((f) => kind === 'all' || f.item.source_type === kind);

  let emptyTitle = 'Nothing discovered yet';
  let emptyBody =
    'No agent has reported a unit, container or cron job yet. Add a service by hand instead.';
  if (hostCount === 0) {
    emptyTitle = 'No hosts yet';
    emptyBody = 'Add a machine and install its agent, or add a service by hand and point it at any address.';
  }

  const tabs = KINDS.map((k) => ({
    key: k.key,
    label: k.label,
    count: matched.filter((f) => k.key === 'all' || f.item.source_type === k.key).length,
  }));

  return (
    <div>
      <PageHeader
        title="Add for monitoring"
        subtitle="Everything the agents have found. Pick one to open a prefilled service, or add a service by hand and point it at any address."
        search={{ value: search, onChange: setSearch, placeholder: 'Search units, containers, cron…' }}
        action={
          <Button variant="secondary" onClick={() => navigate('/services/new')}>
            Add manually
          </Button>
        }
      />

      {found === null && (
        <div className="mt-6">
          <ListSkeleton />
        </div>
      )}

      {found !== null && found.length === 0 && (
        <div className="mt-6">
          <EmptyState
            title={emptyTitle}
            description={emptyBody}
            action={<Button onClick={() => navigate('/services/new')}>Add manually</Button>}
          />
        </div>
      )}

      {found !== null && found.length > 0 && (
        <>
          <div className="mt-6">
            <Tabs tabs={tabs} active={kind} onChange={setKind} label="Filter by kind" />
          </div>
          {shown.length === 0 && (
            <div className="mt-6">
              <EmptyState title="No matches" description="Nothing found matches this search and kind." />
            </div>
          )}
          {shown.length > 0 && (
            <Table
              className="mt-4"
              head={
                <>
                  <Th>Name</Th>
                  <Th>Kind</Th>
                  <Th>Host</Th>
                  <Th>Detail</Th>
                  <Th className="text-right">Action</Th>
                </>
              }
            >
              {shown.map((f) => (
                <Tr key={`${f.host.id}:${f.item.source_type}:${f.item.source_ref}`}>
                  <Td className="font-medium text-content">{f.item.name}</Td>
                  <Td className="whitespace-nowrap text-muted">{KIND_LABEL[f.item.source_type]}</Td>
                  <Td className="whitespace-nowrap">
                    <span className="flex min-w-0 items-center gap-2">
                      <EntityIcon url={f.host.icon_url} name={f.host.name} size={20} />
                      <span className="truncate text-muted">{f.host.name}</span>
                    </span>
                  </Td>
                  <Td className="max-w-xs truncate font-mono text-xs text-muted" title={f.item.detail}>
                    {f.item.detail}
                  </Td>
                  <Td className="text-right">
                    <AddCell found={f} onAdd={() => add(f)} />
                  </Td>
                </Tr>
              ))}
            </Table>
          )}
        </>
      )}

      {found !== null && found.length > 0 && (
        <Section title="Not listed?">
          <Card className="p-4">
            <p className="text-sm text-muted">
              Anything Reeve does not discover — a service on a machine with no agent, a URL somewhere else — is added
              by hand and monitored by its address.
            </p>
            <div className="mt-3">
              <Button variant="secondary" onClick={() => navigate('/services/new')}>
                Add manually
              </Button>
            </div>
          </Card>
        </Section>
      )}
    </div>
  );
}
