import { useState, type FormEvent } from 'react';
import { api, CREDENTIAL_FIELDS, type Credential, type CredentialType } from '../api';
import { Button, ErrorText, Field, Input } from './ui';
import Modal from './Modal';

const TYPES: CredentialType[] = ['ssh_password', 'ssh_key', 'api_token', 'db', 'kv'];

// CredentialForm creates a credential on a tool (rotate reuses the same shape
// via a PATCH, but the UI here is create-only; rotate is a future refinement).
export default function CredentialForm({
  hostId,
  onClose,
  onSaved,
}: {
  hostId: string;
  onClose: () => void;
  onSaved: (c: Credential) => void;
}) {
  const [type, setType] = useState<CredentialType>('ssh_password');
  const [label, setLabel] = useState('');
  const [fields, setFields] = useState<Record<string, string>>({});
  const [kvPairs, setKvPairs] = useState<{ k: string; v: string }[]>([{ k: '', v: '' }]);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    let secret: Record<string, string> = {};
    if (type === 'kv') {
      for (const p of kvPairs) {
        if (p.k.trim()) {
          secret[p.k.trim()] = p.v;
        }
      }
    } else {
      secret = { ...fields };
    }
    if (Object.keys(secret).length === 0) {
      setError('at least one secret field is required');
      return;
    }
    setBusy(true);
    try {
      const c = await api.post<Credential>(`/api/admin/hosts/${hostId}/credentials`, { type, label, secret });
      onSaved(c);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal title="Add credential" onClose={onClose} size="md">
      <form onSubmit={submit} className="mt-4 space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <Field label="Type">
              <select
                value={type}
                onChange={(e) => {
                  setType(e.target.value as CredentialType);
                  setFields({});
                }}
                className="w-full rounded-button border border-hairline-strong bg-canvas px-3 py-2 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-link"
              >
                {TYPES.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Label">
              <Input value={label} onChange={(e) => setLabel(e.target.value)} placeholder="prod root" />
            </Field>
          </div>

          {type !== 'kv' &&
            CREDENTIAL_FIELDS[type].map((f) => (
              <Field key={f} label={f.replace('_', ' ')}>
                {f === 'private_key' ? (
                  <textarea
                    value={fields[f] ?? ''}
                    onChange={(e) => setFields((p) => ({ ...p, [f]: e.target.value }))}
                    rows={4}
                    className="w-full rounded-button border border-hairline-strong bg-canvas px-3 py-2 font-mono text-xs text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-link"
                  />
                ) : (
                  <Input
                    type={f === 'password' ? 'password' : 'text'}
                    value={fields[f] ?? ''}
                    onChange={(e) => setFields((p) => ({ ...p, [f]: e.target.value }))}
                  />
                )}
              </Field>
            ))}

          {type === 'kv' && (
            <div className="space-y-2">
              <span className="text-xs font-medium text-muted">Key / value pairs</span>
              {kvPairs.map((p, i) => (
                <div key={i} className="flex gap-2">
                  <Input
                    placeholder="key"
                    value={p.k}
                    onChange={(e) => setKvPairs((prev) => prev.map((x, j) => (j === i ? { ...x, k: e.target.value } : x)))}
                  />
                  <Input
                    placeholder="value"
                    value={p.v}
                    onChange={(e) => setKvPairs((prev) => prev.map((x, j) => (j === i ? { ...x, v: e.target.value } : x)))}
                  />
                </div>
              ))}
              <Button type="button" variant="secondary" onClick={() => setKvPairs((p) => [...p, { k: '', v: '' }])}>
                Add pair
              </Button>
            </div>
          )}

          <ErrorText>{error}</ErrorText>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={busy}>
              {busy ? 'Saving…' : 'Save credential'}
            </Button>
          </div>
        </form>
    </Modal>
  );
}
