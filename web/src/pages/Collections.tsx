import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, type Collection } from '../api';
import { Button, Card, Pill } from '../components/ui';
import PageHeader from '../components/PageHeader';
import EmptyState from '../components/EmptyState';
import EntityIcon from '../components/EntityIcon';
import CollectionEditor from '../components/CollectionEditor';

export default function Collections() {
  const navigate = useNavigate();
  const [collections, setCollections] = useState<Collection[]>([]);
  const [creating, setCreating] = useState(false);

  const load = useCallback(() => {
    api
      .get<Collection[]>('/api/v1/collections')
      .then((c) => setCollections(c ?? []))
      .catch(() => setCollections([]));
  }, []);
  useEffect(() => {
    load();
  }, [load]);

  return (
    <div>
      <PageHeader
        title="Collections"
        subtitle="Group services so the portal reads by team, not by machine."
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

      <div className="mt-6 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {collections.map((c) => (
          <button key={c.id} type="button" onClick={() => navigate(`/collections/${c.id}`)} className="text-left">
            <Card className="flex h-full flex-col px-4 py-3 transition-colors hover:bg-surface-2">
              <div className="flex items-start justify-between gap-2">
                <EntityIcon url={c.icon_url} name={c.name} size={36} />
                {c.visibility === 'restricted' && <Pill tone="down">restricted</Pill>}
              </div>
              <p className="mt-2 truncate text-sm font-medium text-content">{c.name}</p>
              <p className="truncate text-xs text-muted">{c.description || '—'}</p>
              <p className="mt-2 text-xs text-muted">{c.tool_count} services</p>
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
