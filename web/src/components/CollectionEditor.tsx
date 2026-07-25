import { useCallback, useEffect, useState } from 'react';
import {
  api,
  type Collection,
  type CollectionDetail,
  type Tool,
  type VisibilityGrant,
} from '../api';
import { Button, ErrorText, Field, Input } from './ui';
import EntityIcon from './EntityIcon';
import IconUploader from './IconUploader';
import Modal from './Modal';
import PrincipalPicker from './PrincipalPicker';

const selectCls =
  'w-full rounded-button border border-hairline bg-surface-1 px-3 py-2 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-accent';

// CollectionEditor creates or edits a collection. Grants and the avatar need an
// id, so they only appear once the collection exists.
export default function CollectionEditor({
  collection,
  onClose,
  onSaved,
}: {
  collection?: CollectionDetail;
  onClose: () => void;
  onSaved: (c: Collection) => void;
}) {
  const [name, setName] = useState(collection?.name ?? '');
  const [description, setDescription] = useState(collection?.description ?? '');
  const [visibility, setVisibility] = useState(collection?.visibility ?? 'public');
  const [toolIDs, setToolIDs] = useState<string[]>(collection?.tool_ids ?? []);
  const [tools, setTools] = useState<Tool[]>([]);
  const [iconURL, setIconURL] = useState(collection?.icon_url ?? '');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api
      .get<Tool[]>('/api/v1/tools')
      .then((t) => setTools(t ?? []))
      .catch(() => setTools([]));
  }, []);

  const toggleTool = (id: string) => {
    setToolIDs((p) => {
      if (p.includes(id)) {
        return p.filter((x) => x !== id);
      }
      return [...p, id];
    });
  };

  const syncTools = async (id: string) => {
    const before = collection?.tool_ids ?? [];
    for (const t of toolIDs) {
      if (!before.includes(t)) {
        await api.put(`/api/v1/collections/${id}/tools/${t}`);
      }
    }
    for (const t of before) {
      if (!toolIDs.includes(t)) {
        await api.del(`/api/v1/collections/${id}/tools/${t}`);
      }
    }
  };

  const save = async () => {
    setError('');
    setBusy(true);
    const body = { name: name.trim(), description, visibility };
    try {
      let id = collection?.id ?? '';
      if (collection) {
        await api.patch<Collection>(`/api/v1/collections/${collection.id}`, body);
      } else {
        id = (await api.post<Collection>('/api/v1/collections', body)).id;
      }
      await syncTools(id);
      onSaved(await api.get<Collection>(`/api/v1/collections/${id}`));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed');
      setBusy(false);
    }
  };

  let title = 'New collection';
  if (collection) {
    title = 'Edit collection';
  }

  return (
    <Modal title={title} onClose={onClose} size="lg">
      <div className="mt-4 max-h-[70vh] space-y-4 overflow-y-auto pr-1">
        <Field label="Name">
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Manufacturing" />
        </Field>
        <Field label="Description" hint="Optional. What this collection is for.">
          <textarea
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full rounded-button border border-hairline bg-surface-1 px-3 py-2 text-sm text-content placeholder:text-muted focus:outline-none focus-visible:ring-2 focus-visible:ring-accent"
            placeholder="Shop-floor tooling."
          />
        </Field>

        {collection && (
          <div className="space-y-1.5">
            <span className="text-xs font-medium text-muted">Avatar</span>
            <IconUploader
              url={iconURL}
              name={name}
              path={`/api/v1/collections/${collection.id}/icon`}
              onChange={() =>
                api
                  .get<Collection>(`/api/v1/collections/${collection.id}`)
                  .then((c) => setIconURL(c.icon_url))
                  .catch(() => setIconURL(''))
              }
            />
          </div>
        )}

        <Field label="Visibility" hint="Restricted hides the collection itself, never the services in it.">
          <select
            className={selectCls}
            value={visibility}
            onChange={(e) => setVisibility(e.target.value as Collection['visibility'])}
          >
            <option value="public">public</option>
            <option value="restricted">restricted</option>
          </select>
        </Field>

        {collection && visibility === 'restricted' && (
          <GrantSection label="Who can see it" path={`/api/v1/collections/${collection.id}/visibility`} />
        )}
        {collection && <GrantSection label="Who can edit it" path={`/api/v1/collections/${collection.id}/editors`} />}

        <div className="space-y-1.5">
          <span className="text-xs font-medium text-muted">Services</span>
          <div className="max-h-48 space-y-0.5 overflow-y-auto rounded-button border border-hairline bg-surface-1 p-2">
            {tools.length === 0 && <p className="px-1 py-0.5 text-xs text-muted">No services yet.</p>}
            {tools.map((t) => (
              <label
                key={t.id}
                className="flex cursor-pointer items-center gap-2 rounded-button px-1 py-1 text-sm text-content"
              >
                <input type="checkbox" checked={toolIDs.includes(t.id)} onChange={() => toggleTool(t.id)} />
                <EntityIcon url={t.icon_url} name={t.name} size={18} />
                <span className="truncate">{t.name}</span>
              </label>
            ))}
          </div>
        </div>

        <ErrorText>{error}</ErrorText>
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={save} disabled={busy || !name.trim()}>
            {busy ? 'Saving…' : 'Save'}
          </Button>
        </div>
      </div>
    </Modal>
  );
}

function GrantSection({ label, path }: { label: string; path: string }) {
  const [grants, setGrants] = useState<VisibilityGrant[]>([]);

  const load = useCallback(() => {
    api
      .get<VisibilityGrant[]>(path)
      .then((g) => setGrants(g ?? []))
      .catch(() => setGrants([]));
  }, [path]);
  useEffect(() => {
    load();
  }, [load]);

  const add = async (type: 'user' | 'group', id: string) => {
    await api.put(`${path}/${type}/${id}`);
    load();
  };
  const remove = async (type: 'user' | 'group', id: string) => {
    await api.del(`${path}/${type}/${id}`);
    load();
  };

  return (
    <div className="space-y-1.5">
      <span className="text-xs font-medium text-muted">{label}</span>
      <PrincipalPicker grants={grants} onAdd={add} onRemove={remove} />
    </div>
  );
}
