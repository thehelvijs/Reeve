import { useEffect, useState, type FormEvent, type ReactNode } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { api, type Collection, type Tool } from '../api';
import { Button, ErrorText, Field, Input } from '../components/ui';
import PageHeader from '../components/PageHeader';
import BackLink from '../components/BackLink';
import EntityIcon from '../components/EntityIcon';
import IconUploader from '../components/IconUploader';
import ThumbnailUploader from '../components/ThumbnailUploader';

const SOURCE_TYPES = ['manual', 'systemd', 'docker', 'cron'];

// slugPreview shows what the server will derive from a name, so the field can
// stay empty (and keep tracking the name) until someone deliberately sets one.
// The server does this same derivation; this is only the placeholder.
function slugPreview(name: string): string {
  const slug = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
  return slug || 'grafana';
}

type EndpointMode = 'hostport' | 'url';

// ToolFormPage creates or edits a tool on its own route (/services/new and
// /services/:id/edit) so the form has room to breathe instead of a packed modal.
export default function ToolFormPage() {
  const { id } = useParams();
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const editing = Boolean(id);

  const [tool, setTool] = useState<Tool | null>(null);
  const [loadErr, setLoadErr] = useState('');
  const [ready, setReady] = useState(!editing);

  const [f, setF] = useState({
    name: params.get('name') ?? '',
    description: '',
    tags: '',
    scheme: 'https',
    address: '',
    port: '',
    url: '',
    slug: '',
    source_type: params.get('source_type') ?? 'manual',
    visibility: 'public',
  });
  const [endpoint, setEndpoint] = useState<EndpointMode>('hostport');
  const [logAlert, setLogAlert] = useState(false);
  const [collections, setCollections] = useState<Collection[]>([]);
  const [collectionIDs, setCollectionIDs] = useState<string[]>([]);
  const hostId = tool?.host_id ?? params.get('host_id') ?? '';
  const sourceRef = tool?.source_ref ?? params.get('source_ref') ?? '';
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const set = (k: keyof typeof f, v: string) => setF((p) => ({ ...p, [k]: v }));

  useEffect(() => {
    api
      .get<Collection[]>('/api/collections')
      .then((c) => setCollections(c ?? []))
      .catch(() => setCollections([]));
  }, []);

  useEffect(() => {
    if (!editing) {
      return;
    }
    api
      .get<Tool>(`/api/tools/${id}`)
      .then((t) => {
        setTool(t);
        setCollectionIDs(t.collections.map((c) => c.id));
        setF({
          name: t.name,
          description: t.description,
          tags: t.tags.join(', '),
          scheme: t.scheme || 'https',
          address: t.address,
          port: t.port?.toString() ?? '',
          url: t.url ?? '',
          slug: t.slug,
          source_type: t.source_type,
          visibility: t.visibility,
        });
        setEndpoint(t.url ? 'url' : 'hostport');
        setLogAlert(t.log_alert_enabled);
        setReady(true);
      })
      .catch(() => setLoadErr('Tool not found.'));
  }, [editing, id]);

  const toggleCollection = (cid: string) => {
    setCollectionIDs((p) => {
      if (p.includes(cid)) {
        return p.filter((x) => x !== cid);
      }
      return [...p, cid];
    });
  };
  const addCollection = (c: Collection) => {
    setCollections((p) => [...p, c]);
    setCollectionIDs((p) => [...p, c.id]);
  };

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    const endpointFields =
      endpoint === 'url'
        ? { url: f.url, scheme: '', address: '', port: 0 }
        : { url: '', scheme: f.scheme, address: f.address, port: f.port ? parseInt(f.port, 10) : 0 };
    const body = {
      ...f,
      ...endpointFields,
      host_id: hostId,
      source_ref: sourceRef,
      log_alert_enabled: logAlert,
      collection_ids: collectionIDs,
      tags: f.tags
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean),
    };
    try {
      const saved = editing
        ? await api.patch<Tool>(`/api/tools/${id}`, body)
        : await api.post<Tool>('/api/tools', body);
      navigate(`/services/${saved.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed');
      setBusy(false);
    }
  };

  if (loadErr) {
    return (
      <div>
        <BackLink to="/services">Services</BackLink>
        <p className="mt-4 text-sm text-muted">{loadErr}</p>
      </div>
    );
  }
  if (!ready) {
    return <p className="text-sm text-muted">Loading…</p>;
  }

  return (
    <div className="mx-auto max-w-2xl">
      <BackLink to="/services">Services</BackLink>
      <div className="mt-3">
        <PageHeader
          title={editing ? 'Edit service' : 'Add for monitoring'}
          subtitle="Track whether a service is up and control who can reach it."
        />
      </div>

      <form onSubmit={submit} className="mt-8 space-y-8">
        <Section title="Service">
          <Field label="Name">
            <Input value={f.name} onChange={(e) => set('name', e.target.value)} placeholder="Grafana" required />
          </Field>
          <Field
            label="Short link"
            hint={
              editing
                ? 'Changing this breaks links people have already saved.'
                : 'Optional. Derived from the name when left blank.'
            }
          >
            <div className="flex items-center gap-1.5">
              <span className="shrink-0 font-mono text-xs text-muted">/go/</span>
              <Input
                value={f.slug}
                onChange={(e) => set('slug', e.target.value)}
                placeholder={slugPreview(f.name)}
              />
            </div>
          </Field>
          <Field label="Description" hint="Optional. What this service does.">
            <Input value={f.description} onChange={(e) => set('description', e.target.value)} />
          </Field>
          {editing && id && (
            <>
              <div className="space-y-1.5">
                <span className="text-xs font-medium text-muted">Icon</span>
                <IconUploader
                  url={tool?.icon_url}
                  name={f.name}
                  path={`/api/tools/${id}/icon`}
                  onChange={() => api.get<Tool>(`/api/tools/${id}`).then(setTool)}
                />
              </div>
              <div className="space-y-1.5">
                <span className="text-xs font-medium text-muted">Thumbnail</span>
                <ThumbnailUploader
                  url={tool?.thumbnail_url}
                  path={`/api/tools/${id}/thumbnail`}
                  onChange={() => api.get<Tool>(`/api/tools/${id}`).then(setTool)}
                />
              </div>
            </>
          )}
        </Section>

        <Section title="Endpoint" hint="Where the service is reached.">
          <Segmented
            value={endpoint}
            onChange={setEndpoint}
            options={[
              { value: 'hostport', label: 'Host & port' },
              { value: 'url', label: 'Full URL' },
            ]}
          />
          {endpoint === 'hostport' ? (
            <div className="grid grid-cols-[6rem_1fr_6rem] gap-3">
              <Field label="Scheme">
                <Select value={f.scheme} onChange={(v) => set('scheme', v)} options={['https', 'http']} />
              </Field>
              <Field label="Host / IP">
                <Input value={f.address} onChange={(e) => set('address', e.target.value)} placeholder="10.0.0.5" />
              </Field>
              <Field label="Port">
                <Input value={f.port} onChange={(e) => set('port', e.target.value)} placeholder="3000" />
              </Field>
            </div>
          ) : (
            <Field label="URL">
              <Input value={f.url} onChange={(e) => set('url', e.target.value)} placeholder="https://grafana.lan" />
            </Field>
          )}
        </Section>

        <Section title="Monitoring">
          <Field label="Status source" hint="How up/down is decided. Manual = reachability only.">
            <Select value={f.source_type} onChange={(v) => set('source_type', v)} options={SOURCE_TYPES} />
          </Field>
          <label className="flex items-center gap-2 text-sm text-content">
            <input type="checkbox" checked={logAlert} onChange={(e) => setLogAlert(e.target.checked)} />
            Alert on errors in this service's logs
          </label>
        </Section>

        <Section title="Organize" hint="Optional metadata for the catalog.">
          <div className="grid grid-cols-2 items-start gap-3">
            <CollectionPicker
              collections={collections}
              selected={collectionIDs}
              onToggle={toggleCollection}
              onCreated={addCollection}
            />
            <Field label="Tags" hint="Comma-separated.">
              <Input value={f.tags} onChange={(e) => set('tags', e.target.value)} placeholder="prod, web" />
            </Field>
          </div>
          <div className="max-w-xs">
            <Field label="Visibility" hint="Restricted hides it from the public portal.">
              <Select value={f.visibility} onChange={(v) => set('visibility', v)} options={['public', 'restricted']} />
            </Field>
          </div>
        </Section>

        <ErrorText>{error}</ErrorText>
        <div className="flex justify-end gap-2 border-t border-hairline pt-4">
          <Button type="button" variant="secondary" onClick={() => navigate(editing ? `/services/${id}` : '/services')}>
            Cancel
          </Button>
          <Button type="submit" disabled={busy}>
            {busy ? 'Saving…' : editing ? 'Save changes' : 'Add for monitoring'}
          </Button>
        </div>
      </form>
    </div>
  );
}

function Section({ title, hint, children }: { title: string; hint?: string; children: ReactNode }) {
  return (
    <section className="space-y-3">
      <div>
        <h2 className="text-xs font-medium uppercase tracking-wide text-muted">{title}</h2>
        {hint && <p className="mt-0.5 text-xs text-muted/70">{hint}</p>}
      </div>
      {children}
    </section>
  );
}

function CollectionPicker({
  collections,
  selected,
  onToggle,
  onCreated,
}: {
  collections: Collection[];
  selected: string[];
  onToggle: (id: string) => void;
  onCreated: (c: Collection) => void;
}) {
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const create = async () => {
    setError('');
    setBusy(true);
    try {
      onCreated(await api.post<Collection>('/api/collections', { name: name.trim() }));
      setName('');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-1.5">
      <span className="text-xs font-medium text-muted">Collections</span>
      <div className="max-h-48 space-y-0.5 overflow-y-auto rounded-button border border-hairline bg-surface-1 p-2">
        {collections.length === 0 && <p className="px-1 py-0.5 text-xs text-muted">No collections yet.</p>}
        {collections.map((c) => (
          <label key={c.id} className="flex cursor-pointer items-center gap-2 rounded-button px-1 py-1 text-sm text-content">
            <input type="checkbox" checked={selected.includes(c.id)} onChange={() => onToggle(c.id)} />
            <EntityIcon url={c.icon_url} name={c.name} size={18} />
            <span className="truncate">{c.name}</span>
          </label>
        ))}
      </div>
      <div className="flex gap-2">
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="New collection"
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              if (name.trim() && !busy) {
                create();
              }
            }
          }}
        />
        <Button type="button" variant="secondary" onClick={create} disabled={busy || !name.trim()}>
          Add
        </Button>
      </div>
      <ErrorText>{error}</ErrorText>
    </div>
  );
}

function Segmented<T extends string>({
  value,
  onChange,
  options,
}: {
  value: T;
  onChange: (v: T) => void;
  options: { value: T; label: string }[];
}) {
  return (
    <div className="inline-flex rounded-button border border-hairline p-0.5">
      {options.map((o) => {
        const active = o.value === value;
        return (
          <button
            key={o.value}
            type="button"
            onClick={() => onChange(o.value)}
            className={`rounded-[5px] px-3 py-1 text-sm transition-colors ${
              active ? 'bg-surface-2 text-content' : 'text-muted hover:text-content'
            }`}
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}

function Select({
  value,
  onChange,
  options,
}: {
  value: string;
  onChange: (v: string) => void;
  options: string[];
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="w-full rounded-button border border-hairline bg-surface-1 px-3 py-2 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-accent"
    >
      {options.map((o) => (
        <option key={o} value={o}>
          {o}
        </option>
      ))}
    </select>
  );
}
