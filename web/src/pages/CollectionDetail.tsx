import { useCallback, useEffect, useState } from 'react';
import ConfirmModal from '../components/ConfirmModal';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api, endpointString, type CollectionDetail as Detail, type Tool } from '../api';
import { useAuth } from '../auth';
import { Button, Pill } from '../components/ui';
import Chevron from '../components/Chevron';
import EmptyState from '../components/EmptyState';
import EntityIcon from '../components/EntityIcon';
import StatusPill from '../components/StatusPill';
import CollectionEditor from '../components/CollectionEditor';
import { fmtCount } from '../lib/format';

export default function CollectionDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const [collection, setCollection] = useState<Detail | null>(null);
  const [tools, setTools] = useState<Tool[]>([]);
  const [notFound, setNotFound] = useState(false);
  const [editing, setEditing] = useState(false);

  const load = useCallback(() => {
    api
      .get<Detail>(`/api/collections/${id}`)
      .then(setCollection)
      .catch(() => setNotFound(true));
    api
      .get<Tool[]>(`/api/tools?collection=${id}`)
      .then((t) => setTools(t ?? []))
      .catch(() => setTools([]));
  }, [id]);
  useEffect(() => {
    load();
  }, [load]);

  const [confirming, setConfirming] = useState(false);

  const remove = async () => {
    if (!collection) {
      return;
    }
    await api.del(`/api/collections/${collection.id}`);
    navigate('/collections');
  };

  if (notFound) {
    return <p className="text-sm text-muted">Collection not found.</p>;
  }
  if (!collection) {
    return <p className="text-sm text-muted">Loading…</p>;
  }

  const canDelete = collection.can_edit && (user?.role === 'admin' || user?.id === collection.creator_id);

  return (
    <div>
      <div className="flex items-start justify-between">
        <div className="flex min-w-0 items-start gap-3">
          <EntityIcon url={collection.icon_url} name={collection.name} size={48} />
          <div className="min-w-0">
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-semibold tracking-tight text-content">{collection.name}</h1>
              {collection.visibility === 'restricted' && <Pill tone="down">restricted</Pill>}
            </div>
            {collection.description && <p className="mt-1 text-sm text-muted">{collection.description}</p>}
            <p className="mt-1 text-xs text-muted">{fmtCount(collection.tool_count, 'service')}</p>
          </div>
        </div>
        {collection.can_edit && (
          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => setEditing(true)}>
              Edit
            </Button>
            {canDelete && (
              <Button variant="danger" onClick={() => setConfirming(true)}>
                Delete
              </Button>
            )}
          </div>
        )}
      </div>

      {tools.length > 0 && (
        <div className="mt-6 divide-y divide-hairline overflow-hidden rounded-card border border-hairline">
          {tools.map((t) => (
            <Link
              key={t.id}
              to={`/services/${t.id}`}
              className="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-surface-2"
            >
              <EntityIcon url={t.icon_url} name={t.name} size={36} />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-content">{t.name}</p>
                <p className="truncate font-mono text-xs text-muted">{endpointString(t) || '—'}</p>
              </div>
              <StatusPill status={t.status} />
              <Chevron />
            </Link>
          ))}
        </div>
      )}
      {tools.length === 0 && (
        <div className="mt-6">
          <EmptyState title="No services yet" description="Add services to this collection from its editor." />
        </div>
      )}

      {editing && (
        <CollectionEditor
          collection={collection}
          onClose={() => setEditing(false)}
          onSaved={() => {
            setEditing(false);
            load();
          }}
        />
      )}

      {confirming && (
        <ConfirmModal
          title={`Delete ${collection.name}?`}
          body="The collection is deleted along with its editors and visibility grants. The services in it stay in the catalog."
          confirmLabel="Delete"
          onConfirm={remove}
          onClose={() => setConfirming(false)}
        />
      )}
    </div>
  );
}
