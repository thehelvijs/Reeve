import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { api, endpointString, type Collection, type Tool, type ToolStatus } from '../api';
import { Button, Pill, Table, Tabs, Td, Th, Tr } from '../components/ui';
import StatusPill from '../components/StatusPill';
import EntityIcon from '../components/EntityIcon';
import PageHeader from '../components/PageHeader';
import EmptyState from '../components/EmptyState';
import { ListSkeleton } from '../components/Skeleton';
import VisibilityToggle from '../components/VisibilityToggle';
import { useResource } from '../lib/cache';
import { matchesQuery } from '../lib/search';

// Tab labels say what each state means to an operator, matching the pills.
const STATUS_FILTERS: { key: ToolStatus | 'all'; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'up', label: 'Up' },
  { key: 'down', label: 'Down' },
  { key: 'agent_offline', label: 'Unreachable' },
  { key: 'unknown', label: 'Not monitored' },
];

export default function Catalog() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const [search, setSearch] = useState(params.get('q') ?? '');
  const [status, setStatus] = useState<ToolStatus | 'all'>('all');
  const [collectionID, setCollectionID] = useState('');
  const [collections, setCollections] = useState<Collection[]>([]);

  useEffect(() => {
    setSearch(params.get('q') ?? '');
  }, [params]);

  useEffect(() => {
    api
      .get<Collection[]>('/api/collections')
      .then((c) => setCollections(c ?? []))
      .catch(() => setCollections([]));
  }, []);

  const { data, loading, refresh } = useResource<Tool[]>(
    '/api/tools',
    () => api.get<Tool[]>('/api/tools'),
    15000,
  );
  const tools = useMemo(() => data ?? [], [data]);

  // Filtering is client-side so search is instant and never blanks the list.
  const filtered = useMemo(
    () =>
      tools.filter((t) => {
        if (collectionID && !t.collections.some((c) => c.id === collectionID)) {
          return false;
        }
        return matchesQuery(search, t.name, t.description, t.tags.join(' '));
      }),
    [tools, search, collectionID],
  );

  // Counts come after the search, so a tab counts results rather than everything.
  const shown = filtered.filter((t) => status === 'all' || t.status === status);
  const tabs = STATUS_FILTERS.map((f) => ({
    key: f.key,
    label: f.label,
    count: f.key === 'all' ? filtered.length : filtered.filter((t) => t.status === f.key).length,
  }));

  return (
    <div>
      <PageHeader
        title="Services"
        search={{ value: search, onChange: setSearch, placeholder: 'Search services…' }}
        action={
          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => navigate('/services/new')}>
              Add manually
            </Button>
            <Button onClick={() => navigate('/services/add')}>Add for monitoring</Button>
          </div>
        }
      />

      <div className="mt-6 flex gap-3">
        <select
          aria-label="Filter by collection"
          value={collectionID}
          onChange={(e) => setCollectionID(e.target.value)}
          className="rounded-button border border-hairline-strong bg-canvas px-3 py-2 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-link"
        >
          <option value="">All collections</option>
          {collections.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
      </div>

      <div className="mt-4">
        <Tabs tabs={tabs} active={status} onChange={setStatus} label="Filter services by state" />
      </div>

      {loading ? (
        <div className="mt-6">
          <ListSkeleton />
        </div>
      ) : (
        <>
          {shown.length > 0 && (
            <Table
              className="mt-4"
              head={
                <>
                  <Th>Service</Th>
                  <Th>Endpoint</Th>
                  <Th>Collections</Th>
                  <Th>Who can see it</Th>
                  <Th>State</Th>
                  <Th className="text-right">Actions</Th>
                </>
              }
            >
              {shown.map((t) => (
                <Tr key={t.id} to={`/services/${t.id}`}>
                  <Td>
                    <div className="flex min-w-0 items-center gap-2.5">
                      <EntityIcon url={t.icon_url} name={t.name} size={28} />
                      <Link
                        to={`/services/${t.id}`}
                        className="truncate font-medium text-link hover:text-link-hover hover:underline"
                      >
                        {t.name}
                      </Link>
                    </div>
                  </Td>
                  <Td className="font-mono text-xs text-muted">{endpointString(t) || '—'}</Td>
                  <Td>
                    {t.collections.length > 0 ? (
                      <div className="flex flex-wrap gap-1.5">
                        {t.collections.map((c) => (
                          <Pill key={c.id}>{c.name}</Pill>
                        ))}
                      </div>
                    ) : (
                      <span className="text-muted">—</span>
                    )}
                  </Td>
                  <Td>
                    <VisibilityToggle tool={t} onChanged={refresh} />
                  </Td>
                  <Td>
                    <StatusPill status={t.status} />
                  </Td>
                  <Td className="text-right">
                    {t.can_edit && (
                      <button
                        type="button"
                        onClick={() => navigate(`/services/${t.id}/edit`)}
                        className="inline-flex shrink-0 items-center rounded-button border border-hairline-strong bg-canvas px-2 py-0.5 text-xs text-content transition-colors hover:bg-surface-2 focus:outline-none focus-visible:ring-2 focus-visible:ring-link"
                      >
                        Edit
                      </button>
                    )}
                  </Td>
                </Tr>
              ))}
            </Table>
          )}
          {shown.length === 0 && (
            <div className="mt-6">
              {tools.length === 0 ? (
                <EmptyState
                  title="No services yet"
                  description="Add a service to track its state and set who can reach it."
                  action={
          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => navigate('/services/new')}>
              Add manually
            </Button>
            <Button onClick={() => navigate('/services/add')}>Add for monitoring</Button>
          </div>
        }
                />
              ) : (
                <EmptyState
                  title="No matches"
                  description="No service matches this search, collection and state."
                />
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
