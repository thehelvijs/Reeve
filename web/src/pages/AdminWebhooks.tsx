import { useEffect, useState } from 'react';
import ConfirmModal from '../components/ConfirmModal';
import { api, type Severity, type Webhook } from '../api';
import { Button, Card, ErrorText, Field, Form, Input, Pill } from '../components/ui';
import PageHeader from '../components/PageHeader';
import { matchesQuery } from '../lib/search';

const selectClass =
  'rounded-button border border-hairline-strong bg-canvas px-3 py-2 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-link';

export default function AdminWebhooks() {
  const [hooks, setHooks] = useState<Webhook[]>([]);
  const [ownerType, setOwnerType] = useState<'global' | 'tool' | 'group'>('global');
  const [ownerId, setOwnerId] = useState('');
  const [url, setUrl] = useState('');
  const [token, setToken] = useState('');
  const [minSeverity, setMinSeverity] = useState<Severity>('info');
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  // One result per row plus one for the form, keyed by webhook id and 'new' for
  // the unsaved one, so testing a second channel does not clear the first verdict.
  const [tested, setTested] = useState<Record<string, string>>({});
  const [testing, setTesting] = useState<string | null>(null);

  // A saved channel tests by id: its token is redacted in every read, so only the
  // server can put the real one on the wire.
  const test = async (key: string, body: Record<string, unknown>) => {
    setTesting(key);
    setTested((t) => ({ ...t, [key]: '' }));
    try {
      const out = await api.post<{ ok: boolean; error?: string }>('/api/admin/webhooks/test', body);
      if (out.ok) {
        setTested((t) => ({ ...t, [key]: 'delivered' }));
      } else {
        setTested((t) => ({ ...t, [key]: out.error || 'the receiver rejected it' }));
      }
    } catch (e) {
      setTested((t) => ({ ...t, [key]: e instanceof Error ? e.message : 'test failed' }));
    }
    setTesting(null);
  };

  const testDraft = () => {
    const config: Record<string, string> = {};
    if (token.trim()) {
      config.token = token.trim();
    }
    return test('new', { url: url.trim(), config });
  };

  const load = () => api.get<Webhook[]>('/api/admin/webhooks').then((h) => setHooks(h ?? []));
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
      await api.post('/api/admin/webhooks', {
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
  const [confirming, setConfirming] = useState<Webhook | null>(null);

  const remove = async (id: string) => {
    await api.del(`/api/admin/webhooks/${id}`);
    load();
  };

  const shown = hooks.filter((h) => matchesQuery(search, h.url, h.owner_type, h.owner_id, h.min_severity));

  return (
    <div>
      <PageHeader
        title="Webhooks"
        search={{ value: search, onChange: setSearch, placeholder: 'Search webhooks…' }}
      />

      <Card className="mt-6 p-5">
        <Form onSubmit={create}>
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

          <div className="mt-4 flex items-center justify-between gap-3">
            <p className="text-xs text-muted">The token is encrypted at rest and never shown again after saving.</p>
            <div className="flex shrink-0 items-center gap-2">
              <TestResult result={tested.new} />
              <Button
                type="button"
                variant="secondary"
                disabled={!url.trim() || testing === 'new'}
                onClick={testDraft}
              >
                {testing === 'new' ? 'Testing…' : 'Test'}
              </Button>
              <Button type="submit" disabled={!url.trim()}>
                Add webhook
              </Button>
            </div>
          </div>
          <ErrorText>{error}</ErrorText>
        </Form>
      </Card>

      <div className="mt-6 space-y-2">
        {hooks.length === 0 && <p className="text-sm text-muted">No webhooks configured.</p>}
        {hooks.length > 0 && shown.length === 0 && (
          <p className="text-sm text-muted">No webhook matches the search.</p>
        )}
        {shown.map((h) => (
          <Card key={h.id} className="flex items-center justify-between px-4 py-3">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <Pill>{h.owner_type}</Pill>
                <Pill>{h.min_severity}</Pill>
                {h.owner_id && <span className="text-xs text-muted">{h.owner_id}</span>}
              </div>
              {h.url && <p className="mt-1 truncate font-mono text-xs text-muted">{h.url}</p>}
            </div>
            <div className="flex shrink-0 items-center gap-2">
              <TestResult result={tested[h.id]} />
              <Button
                variant="secondary"
                disabled={testing === h.id}
                onClick={() => test(h.id, { id: h.id })}
              >
                {testing === h.id ? 'Testing…' : 'Test'}
              </Button>
              <Button variant="danger" onClick={() => setConfirming(h)}>
                Delete
              </Button>
            </div>
          </Card>
        ))}
      </div>

      {confirming && (
        <ConfirmModal
          title="Delete this webhook?"
          body={`Alerts stop being delivered to ${confirming.url || 'it'}. Nothing already delivered is affected.`}
          confirmLabel="Delete"
          onConfirm={() => remove(confirming.id)}
          onClose={() => setConfirming(null)}
        />
      )}
    </div>
  );
}

// The verdict sits beside the button that produced it: a receiver that answers
// badly is the common case, and its own words are more use than "failed".
function TestResult({ result }: { result?: string }) {
  if (!result) {
    return null;
  }
  if (result === 'delivered') {
    return <span className="text-xs text-up">Delivered</span>;
  }
  return <span className="max-w-64 truncate text-xs text-down" title={result}>{result}</span>;
}
