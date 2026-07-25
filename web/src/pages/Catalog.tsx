import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { api, endpointString, type Collection, type Tool } from '../api';
import { Button, Input, Pill } from '../components/ui';
import StatusPill from '../components/StatusPill';
import EntityIcon from '../components/EntityIcon';
import Chevron from '../components/Chevron';
import PageHeader from '../components/PageHeader';
import EmptyState from '../components/EmptyState';
import { ListSkeleton } from '../components/Skeleton';
import { useResource } from '../lib/cache';

export default function Catalog() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const [search, setSearch] = useState(params.get('q') ?? '');
  const [collectionID, setCollectionID] = useState('');
  const [collections, setCollections] = useState<Collection[]>([]);

  useEffect(() => {
    setSearch(params.get('q') ?? '');
  }, [params]);

  useEffect(() => {
    api
      .get<Collection[]>('/api/v1/collections')
      .then((c) => setCollections(c ?? []))
      .catch(() => setCollections([]));
  }, []);

  const { data, loading } = useResource<Tool[]>(
    '/api/v1/tools',
    () => api.get<Tool[]>('/api/v1/tools'),
    15000,
  );
  const tools = useMemo(() => data ?? [], [data]);

  // Filtering is client-side so search is instant and never blanks the list.
  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return tools.filter((t) => {
      if (collectionID && !t.collections.some((c) => c.id === collectionID)) {
        return false;
      }
      if (!q) {
        return true;
      }
      return (
        t.name.toLowerCase().includes(q) ||
        t.description.toLowerCase().includes(q) ||
        t.tags.some((tag) => tag.toLowerCase().includes(q))
      );
    });
  }, [tools, search, collectionID]);

  return (
    <div>
      <PageHeader
        title="Services"
        subtitle="Every service, where it lives, and whether it's up."
        action={<Button onClick={() => navigate('/catalog/new')}>Add for monitoring</Button>}
      />

      <div className="mt-6 flex gap-3">
        <Input
          placeholder="Search tools…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-xs"
        />
        <select
          value={collectionID}
          onChange={(e) => setCollectionID(e.target.value)}
          className="rounded-button border border-hairline bg-surface-1 px-3 py-2 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-accent"
        >
          <option value="">All collections</option>
          {collections.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
      </div>

      {loading ? (
        <div className="mt-6">
          <ListSkeleton />
        </div>
      ) : (
        <>
          <div className="mt-6 divide-y divide-hairline overflow-hidden rounded-card border border-hairline">
            {filtered.map((t) => (
              <Link
                key={t.id}
                to={`/catalog/${t.id}`}
                className="group flex items-center gap-3 px-4 py-3 transition-colors hover:bg-surface-2"
              >
                <EntityIcon url={t.icon_url} name={t.name} size={36} />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-content">{t.name}</p>
                  <p className="truncate font-mono text-xs text-muted">{endpointString(t) || '—'}</p>
                  {(t.collections.length > 0 || t.visibility === 'restricted') && (
                    <div className="mt-1.5 flex flex-wrap gap-1.5">
                      {t.collections.map((c) => (
                        <Pill key={c.id}>{c.name}</Pill>
                      ))}
                      {t.visibility === 'restricted' && <Pill tone="down">restricted</Pill>}
                    </div>
                  )}
                </div>
                <StatusPill status={t.status} />
                {t.can_edit && (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      navigate(`/catalog/${t.id}/edit`);
                    }}
                    className="rounded-button border border-hairline px-2 py-1 text-xs text-muted opacity-0 transition-opacity hover:text-content group-hover:opacity-100"
                  >
                    Edit
                  </button>
                )}
                <Chevron />
              </Link>
            ))}
          </div>
          {filtered.length === 0 && (
            <div className="mt-6">
              {tools.length === 0 ? (
                <EmptyState
                  title="No services yet"
                  description="Add a service to track whether it's up and control who can reach it."
                  action={<Button onClick={() => navigate('/catalog/new')}>Add for monitoring</Button>}
                />
              ) : (
                <EmptyState title="No matches" description="No services match your search or collection." />
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
