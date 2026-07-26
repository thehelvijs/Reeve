import { useCallback, useEffect, useState } from 'react';
import { api, type AccessRequest, type Credential, type RevealedCredential, type Tool } from '../api';
import { Button, Card } from './ui';
import CredentialForm from './CredentialForm';
import RevealModal from './RevealModal';

export default function CredentialsSection({ tool }: { tool: Tool }) {
  const [creds, setCreds] = useState<Credential[]>([]);
  const [adding, setAdding] = useState(false);
  const [revealed, setRevealed] = useState<RevealedCredential | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(() => {
    api.get<Credential[]>(`/api/tools/${tool.id}/credentials`).then((c) => setCreds(c ?? []));
    api.get<AccessRequest[]>('/api/access-requests?box=mine').then((rs) => {
      setPending((rs ?? []).some((r) => r.tool_id === tool.id && r.status === 'pending'));
    });
  }, [tool.id]);
  useEffect(() => {
    load();
  }, [load]);

  const reveal = async (id: string) => {
    setError('');
    try {
      setRevealed(await api.post<RevealedCredential>(`/api/credentials/${id}/reveal`));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'reveal failed');
    }
  };
  const remove = async (id: string) => {
    await api.del(`/api/credentials/${id}`);
    load();
  };
  const requestAccess = async () => {
    await api.post(`/api/tools/${tool.id}/access-requests`, { note: '' });
    setPending(true);
  };

  const canReveal = creds.length > 0 && creds[0].can_reveal;
  const needsAccess = creds.length > 0 && !canReveal && !tool.can_edit;

  return (
    <Card className="mt-6 p-5">
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium text-content">Credentials</p>
        {tool.can_edit && (
          <Button variant="secondary" onClick={() => setAdding(true)}>
            Add credential
          </Button>
        )}
        {needsAccess &&
          (pending ? (
            <span className="text-xs text-muted">Access requested</span>
          ) : (
            <Button onClick={requestAccess}>Request access</Button>
          ))}
      </div>

      {creds.length === 0 && <p className="mt-3 text-sm text-muted">No credentials stored.</p>}

      <div className="mt-3 space-y-2">
        {creds.map((c) => (
          <div
            key={c.id}
            className="flex items-center justify-between rounded-button border border-hairline bg-surface-2 px-3 py-2"
          >
            <div>
              <span className="text-sm text-content">{c.label || c.type}</span>
              <span className="ml-2 text-xs text-muted">{c.type}</span>
            </div>
            <div className="flex gap-2">
              {c.can_reveal && (
                <Button variant="secondary" onClick={() => reveal(c.id)}>
                  Reveal
                </Button>
              )}
              {tool.can_edit && (
                <Button variant="danger" onClick={() => remove(c.id)}>
                  Delete
                </Button>
              )}
            </div>
          </div>
        ))}
      </div>
      {error && <p className="mt-2 text-sm text-red-400">{error}</p>}

      {adding && (
        <CredentialForm
          toolId={tool.id}
          onClose={() => setAdding(false)}
          onSaved={() => {
            setAdding(false);
            load();
          }}
        />
      )}
      {revealed && <RevealModal cred={revealed} onClose={() => setRevealed(null)} />}
    </Card>
  );
}
