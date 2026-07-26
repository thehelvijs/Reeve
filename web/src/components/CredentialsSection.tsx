import { useCallback, useEffect, useState } from 'react';
import { api, type AccessRequest, type Credential, type RevealedCredential } from '../api';
import { Button, Card } from './ui';
import CredentialForm from './CredentialForm';
import RevealModal from './RevealModal';

// CredentialsSection lists how to get into a host. Credentials belong to the
// machine, not to a service running on it, so this lives on the host page.
// canManage is admin-only; revealing is a separate right a grant can carry.
export default function CredentialsSection({
  hostId,
  canManage,
}: {
  hostId: string;
  canManage: boolean;
}) {
  const [creds, setCreds] = useState<Credential[]>([]);
  const [adding, setAdding] = useState(false);
  const [revealed, setRevealed] = useState<RevealedCredential | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(() => {
    api.get<Credential[]>(`/api/hosts/${hostId}/credentials`).then((c) => setCreds(c ?? []));
    api.get<AccessRequest[]>('/api/access-requests?box=mine').then((rs) => {
      setPending((rs ?? []).some((r) => r.host_id === hostId && r.status === 'pending'));
    });
  }, [hostId]);
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
    await api.del(`/api/admin/credentials/${id}`);
    load();
  };
  const requestAccess = async () => {
    setError('');
    try {
      await api.post(`/api/hosts/${hostId}/access-requests`, { note: '' });
      setPending(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not request access');
    }
  };

  const canReveal = creds.length > 0 && creds[0].can_reveal;
  const needsAccess = creds.length > 0 && !canReveal && !canManage;

  return (
    <Card className="mt-4 p-5">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-content">Credentials</p>
          <p className="mt-1 text-xs text-muted">How to connect to this machine.</p>
        </div>
        {canManage && (
          <Button variant="secondary" onClick={() => setAdding(true)}>
            Add credential
          </Button>
        )}
        {needsAccess && !pending && <Button onClick={requestAccess}>Request access</Button>}
        {needsAccess && pending && <span className="text-xs text-muted">Access requested</span>}
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
              {canManage && (
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
          hostId={hostId}
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
