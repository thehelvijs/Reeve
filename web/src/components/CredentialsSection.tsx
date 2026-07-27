import { useCallback, useEffect, useState } from 'react';
import ConfirmModal from './ConfirmModal';
import {
  api,
  CREDENTIAL_LABEL,
  type AccessRequest,
  type Credential,
  type RevealedCredential,
} from '../api';
import { Button, ErrorText, Section, Table, Td, Th, Tr } from './ui';
import EmptyState from './EmptyState';
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
  const [confirming, setConfirming] = useState<Credential | null>(null);

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

  const addButton = <Button variant="secondary" onClick={() => setAdding(true)}>Add credential</Button>;
  let action = null;
  if (canManage) {
    action = addButton;
  } else if (needsAccess && !pending) {
    action = <Button onClick={requestAccess}>Request access</Button>;
  } else if (needsAccess && pending) {
    action = <span className="text-xs text-muted">Access requested</span>;
  }

  let empty = 'Nobody has stored a login for this machine. An admin can add one.';
  if (canManage) {
    empty = 'Store the login for this machine here, and it is encrypted at rest.';
  }

  return (
    <Section
      title="Credentials"
      count={creds.length}
      description="How to connect to this machine. Every reveal is recorded against the host."
      action={action}
    >
      {creds.length === 0 ? (
        <EmptyState title="No credentials stored" description={empty} />
      ) : (
        <Table
          head={
            <>
              <Th>Credential</Th>
              <Th>Type</Th>
              <Th className="text-right">Actions</Th>
            </>
          }
        >
          {creds.map((c) => (
            <Tr key={c.id}>
              <Td className="font-medium text-content">{c.label || CREDENTIAL_LABEL[c.type]}</Td>
              <Td className="whitespace-nowrap text-muted">{CREDENTIAL_LABEL[c.type]}</Td>
              <Td className="text-right">
                <div className="flex items-center justify-end gap-2">
                  {c.can_reveal && (
                    <Button variant="secondary" onClick={() => reveal(c.id)}>
                      Reveal
                    </Button>
                  )}
                  {canManage && (
                    <Button variant="danger" onClick={() => setConfirming(c)}>
                      Delete
                    </Button>
                  )}
                </div>
              </Td>
            </Tr>
          ))}
        </Table>
      )}
      <ErrorText>{error}</ErrorText>

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
      {confirming && (
        <ConfirmModal
          title={`Delete ${confirming.label || CREDENTIAL_LABEL[confirming.type]}?`}
          body="The stored secret is deleted and anyone holding a grant for it loses access. Whoever needs it again has to add it back by hand."
          confirmLabel="Delete"
          onConfirm={() => remove(confirming.id)}
          onClose={() => setConfirming(null)}
        />
      )}
      {revealed && <RevealModal cred={revealed} onClose={() => setRevealed(null)} />}
    </Section>
  );
}
