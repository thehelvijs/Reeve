import { useEffect, useState } from 'react';
import ConfirmModal from '../components/ConfirmModal';
import { api, type AdminUser } from '../api';
import { useAuth } from '../auth';
import { Button, Card, ErrorText, Field, Input, Pill } from '../components/ui';
import PageHeader from '../components/PageHeader';
import Avatar from '../components/Avatar';
import Modal from '../components/Modal';
import { matchesQuery } from '../lib/search';

export default function AdminUsers() {
  const { user: me } = useAuth();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [resetting, setResetting] = useState<AdminUser | null>(null);
  const [search, setSearch] = useState('');

  const load = () => api.get<AdminUser[]>('/api/admin/users').then((u) => setUsers(u ?? []));
  useEffect(() => {
    load();
  }, []);

  const update = async (id: string, patch: { role?: string; active?: boolean }) => {
    await api.patch(`/api/admin/users/${id}`, patch);
    load();
  };

  const [confirming, setConfirming] = useState<AdminUser | null>(null);

  const remove = async (u: AdminUser) => {
    await api.del(`/api/admin/users/${u.id}`);
    load();
  };

  const shown = users.filter((u) => matchesQuery(search, u.email, u.display_name, u.role));

  return (
    <div>
      <PageHeader
        title="Users"
        search={{ value: search, onChange: setSearch, placeholder: 'Search users…' }}
      />

      <div className="mt-6 space-y-2">
        {users.length === 0 && <p className="text-sm text-muted">No users yet.</p>}
        {users.length > 0 && shown.length === 0 && (
          <p className="text-sm text-muted">No user matches the search.</p>
        )}
        {shown.map((u) => {
          const self = u.id === me?.id;
          return (
            <Card key={u.id} className="flex items-center justify-between px-4 py-3">
              <div className="flex min-w-0 items-center gap-3">
                <Avatar url={u.avatar_url} name={u.display_name} email={u.email} size={32} />
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="truncate text-sm text-content">{u.display_name || u.email}</span>
                    <Pill tone={u.role === 'admin' ? 'up' : 'muted'}>{u.role}</Pill>
                    {!u.active && <Pill tone="down">inactive</Pill>}
                  </div>
                  {u.display_name && <p className="truncate text-xs text-muted">{u.email}</p>}
                </div>
              </div>
              <div className="flex shrink-0 gap-2">
                <Button
                  variant="secondary"
                  disabled={self}
                  onClick={() => update(u.id, { role: u.role === 'admin' ? 'basic' : 'admin' })}
                >
                  {u.role === 'admin' ? 'Make basic' : 'Make admin'}
                </Button>
                <Button
                  variant={u.active ? 'danger' : 'secondary'}
                  disabled={self}
                  onClick={() => update(u.id, { active: !u.active })}
                >
                  {u.active ? 'Deactivate' : 'Activate'}
                </Button>
                <Button variant="secondary" disabled={self} onClick={() => setResetting(u)}>
                  Reset password
                </Button>
                <Button variant="danger" disabled={self} onClick={() => setConfirming(u)}>
                  Delete
                </Button>
              </div>
            </Card>
          );
        })}
      </div>

      {confirming && (
        <ConfirmModal
          title={`Delete ${confirming.email}?`}
          body="The account is deleted and their services and credentials reassign to you. Their sessions end immediately. This cannot be undone."
          confirmLabel="Delete"
          onConfirm={() => remove(confirming)}
          onClose={() => setConfirming(null)}
        />
      )}
      {resetting && <ResetPasswordModal user={resetting} onClose={() => setResetting(null)} />}
    </div>
  );
}

function ResetPasswordModal({ user, onClose }: { user: AdminUser; onClose: () => void }) {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);

  const submit = async () => {
    setBusy(true);
    setError('');
    try {
      await api.post(`/api/admin/users/${user.id}/password`, { password });
      setDone(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'reset failed');
    }
    setBusy(false);
  };

  if (done) {
    return (
      <Modal title="Password reset" onClose={onClose} size="md" onSubmit={onClose}>
        <p className="mt-2 text-sm text-muted">
          {user.email} is signed out everywhere and must use the new password. Hand it over out of band.
        </p>
        <div className="mt-5 flex justify-end">
          <Button onClick={onClose}>Done</Button>
        </div>
      </Modal>
    );
  }

  const submitAction = () => {
    if (busy || password === '') {
      return undefined;
    }
    return submit;
  };

  return (
    <Modal title="Reset password" onClose={onClose} size="md" onSubmit={submitAction()}>
      <p className="mt-2 text-sm text-muted">{user.email} will be signed out everywhere.</p>
      <div className="mt-4">
        <Field label="New password">
          <Input
            type="text"
            value={password}
            autoComplete="new-password"
            onChange={(e) => setPassword(e.target.value)}
          />
        </Field>
      </div>
      <div className="mt-3">
        <ErrorText>{error}</ErrorText>
      </div>
      <div className="mt-5 flex justify-end gap-2">
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        <Button disabled={busy || password === ''} onClick={submit}>
          {busy ? 'Resetting…' : 'Reset password'}
        </Button>
      </div>
    </Modal>
  );
}
