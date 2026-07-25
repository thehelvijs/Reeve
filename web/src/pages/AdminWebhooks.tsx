import { useEffect, useState } from 'react';
import { api, type Severity, type Webhook } from '../api';
import { Button, Card, ErrorText, Field, Input, Pill } from '../components/ui';

const selectClass =
  'rounded-button border border-hairline bg-surface-1 px-3 py-2 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-accent';

export default function AdminWebhooks() {
  const [hooks, setHooks] = useState<Webhook[]>([]);
  const [ownerType, setOwnerType] = useState<'global' | 'tool' | 'group'>('global');
  const [ownerId, setOwnerId] = useState('');
  const [url, setUrl] = useState('');
  const [token, setToken] = useState('');
  const [minSeverity, setMinSeverity] = useState<Severity>('info');
  const [error, setError] = useState('');

  const load = () => api.get<Webhook[]>('/api/v1/admin/webhooks').then((h) => setHooks(h ?? []));
  useEffect(() => {
    load();
  }, []);

  const create = async () => {
    setError('');
    const config: Record<string, string> = {};
    if (token.trim()) {
      config.token = token.trim();
    }
    try {
      await api.post('/api/v1/admin/webhooks', {
        owner_type: ownerType,
        owner_id: ownerId,
        url,
        format: 'webhook',
        min_severity: minSeverity,
        config,
      });
      setUrl('');
      setOwnerId('');
      setToken('');
      setMinSeverity('info');
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed');
    }
  };
  const remove = async (id: string) => {
    await api.del(`/api/v1/admin/webhooks/${id}`);
    load();
  };

  return (
    <div>
      <h1 className="text-2xl font-semibold tracking-tight text-content">Webhooks</h1>
      <p className="mt-1 text-sm text-muted">
        Alerts POST a JSON payload to these URLs. Point them at any HTTP endpoint on your network.
      </p>

      <Card className="mt-6 p-5">
        <div className="flex flex-wrap items-end gap-3">
          <Field label="Scope">
            <select value={ownerType} onChange={(e) => setOwnerType(e.target.value as typeof ownerType)} className={selectClass}>
              <option value="global">global</option>
              <option value="tool">tool</option>
              <option value="group">group</option>
            </select>
          </Field>
          {ownerType !== 'global' && (
            <Field label={`${ownerType} id`}>
              <Input value={ownerId} onChange={(e) => setOwnerId(e.target.value)} />
            </Field>
          )}
          <Field label="Min severity">
            <select value={minSeverity} onChange={(e) => setMinSeverity(e.target.value as Severity)} className={selectClass}>
              <option value="info">info</option>
              <option value="warning">warning</option>
              <option value="error">error</option>
            </select>
          </Field>
        </div>

        <div className="mt-3">
          <Field label="URL">
            <Input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://sink.example.com/hook" />
          </Field>
        </div>
        <div className="mt-3">
          <Field label="Bearer token (optional)">
            <Input type="password" value={token} onChange={(e) => setToken(e.target.value)} />
          </Field>
        </div>

        <div className="mt-4 flex items-center justify-between">
          <p className="text-xs text-muted">The token is encrypted at rest and never shown again after saving.</p>
          <Button onClick={create} disabled={!url.trim()}>
            Add webhook
          </Button>
        </div>
        <ErrorText>{error}</ErrorText>
      </Card>

      <div className="mt-6 space-y-2">
        {hooks.length === 0 && <p className="text-sm text-muted">No webhooks configured.</p>}
        {hooks.map((h) => (
          <Card key={h.id} className="flex items-center justify-between px-4 py-3">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <Pill>{h.owner_type}</Pill>
                <Pill>{h.min_severity}</Pill>
                {h.owner_id && <span className="text-xs text-muted">{h.owner_id}</span>}
              </div>
              {h.url && <p className="mt-1 truncate font-mono text-xs text-muted">{h.url}</p>}
            </div>
            <Button variant="danger" onClick={() => remove(h.id)}>
              Delete
            </Button>
          </Card>
        ))}
      </div>
    </div>
  );
}
