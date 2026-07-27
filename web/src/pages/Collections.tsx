import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, type Collection } from '../api';
import { Button, Card, Pill } from '../components/ui';
import PageHeader from '../components/PageHeader';
import EmptyState from '../components/EmptyState';
import EntityIcon from '../components/EntityIcon';
import CollectionEditor from '../components/CollectionEditor';
import { fmtCount } from '../lib/format';
import { matchesQuery } from '../lib/search';

export default function Collections() {
  const navigate = useNavigate();
  const [collections, setCollections] = useState<Collection[]>([]);
  const [creating, setCreating] = useState(false);
  const [search, setSearch] = useState('');

  const load = useCallback(() => {
    api
      .get<Collection[]>('/api/collections')
      .then((c) => setCollections(c ?? []))
      .catch(() => setCollections([]));
  }, []);
  useEffect(() => {
    load();
  }, [load]);

  const shown = collections.filter((c) => matchesQuery(search, c.name, c.description));

  return (
    <div>
      <PageHeader
        title="Collections"
        subtitle="Group services so the portal reads by team, not by machine."
        search={{ value: search, onChange: setSearch, placeholder: 'Search collections…' }}
        action={<Button onClick={() => setCreating(true)}>New collection</Button>}
      />

      {collections.length === 0 && (
        <div className="mt-6">
          <EmptyState
            title="No collections yet"
            description="Create a collection to group services on the portal by team instead of by host."
            action={<Button onClick={() => setCreating(true)}>New collection</Button>}
          />
        </div>
      )}

      {collections.length > 0 && shown.length === 0 && (
        <p className="mt-6 text-sm text-muted">No collection matches the search.</p>
      )}

      <div className="mt-6 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {shown.map((c) => (
          <button key={c.id} type="button" onClick={() => navigate(`/collections/${c.id}`)} className="text-left">
            <Card className="flex h-full flex-col px-4 py-3 transition-colors hover:bg-surface-2">
              <div className="flex items-start justify-between gap-2">
                <EntityIcon url={c.icon_url} name={c.name} size={36} />
                {c.visibility === 'restricted' && <Pill tone="down">restricted</Pill>}
              </div>
              <p className="mt-2 truncate text-sm font-medium text-content">{c.name}</p>
              <p className="truncate text-xs text-muted">{c.description || '—'}</p>
              <p className="mt-2 text-xs text-muted">{fmtCount(c.tool_count, 'service')}</p>
            </Card>
          </button>
        ))}
      </div>

      {creating && (
        <CollectionEditor
          onClose={() => setCreating(false)}
          onSaved={(c) => {
            setCreating(false);
            navigate(`/collections/${c.id}`);
          }}
        />
      )}
    </div>
  );
}
