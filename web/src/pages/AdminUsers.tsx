import { useEffect, useState } from 'react';
import ConfirmModal from '../components/ConfirmModal';
import { api, type AdminUser, type Invite } from '../api';
import { useAuth } from '../auth';
import { Button, Card, ErrorText, Field, Input, Pill, Select } from '../components/ui';
import PageHeader from '../components/PageHeader';
import Avatar from '../components/Avatar';
import Modal from '../components/Modal';
import { matchesQuery } from '../lib/search';

export default function AdminUsers() {
  const { user: me } = useAuth();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [resetting, setResetting] = useState<AdminUser | null>(null);
  const [inviting, setInviting] = useState(false);
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
        action={<Button onClick={() => setInviting(true)}>Invite user</Button>}
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
      {inviting && (
        <InviteModal
          onClose={() => setInviting(false)}
          onInvited={load}
        />
      )}
    </div>
  );
}

// An invite creates the account and issues the link that sets its first password.
// With a relay configured that link is mailed and never shown; without one it is
// shown here, because otherwise the account has no way in.
function InviteModal({ onClose, onInvited }: { onClose: () => void; onInvited: () => void }) {
  const [email, setEmail] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [role, setRole] = useState('basic');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<Invite | null>(null);
  const [copied, setCopied] = useState(false);

  const submit = async () => {
    setBusy(true);
    setError('');
    try {
      const out = await api.post<Invite>('/api/admin/users', {
        email,
        role,
        display_name: displayName,
      });
      setResult(out);
      onInvited();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'invite failed');
    }
    setBusy(false);
  };

  const copy = async () => {
    await navigator.clipboard.writeText(result?.invite_link ?? '');
    setCopied(true);
  };

  if (result) {
    return (
      <Modal title="Invite sent" onClose={onClose} size="md" onSubmit={onClose}>
        {result.emailed ? (
          <p className="mt-2 text-sm text-muted">
            {result.user.email} has an email with a link to choose a password. It expires in seven days.
          </p>
        ) : (
          <>
            <p className="mt-2 text-sm text-muted">
              No mail relay is configured, so nothing was sent. Hand this link to {result.user.email}. It
              sets their password and expires in seven days.
            </p>
            <p className="mt-3 break-all rounded-button border border-hairline bg-surface-1 p-3 font-mono text-xs text-content">
              {result.invite_link}
            </p>
            <div className="mt-3">
              <Button variant="secondary" onClick={copy}>
                {copied ? 'Copied' : 'Copy link'}
              </Button>
            </div>
          </>
        )}
        <div className="mt-5 flex justify-end">
          <Button onClick={onClose}>Done</Button>
        </div>
      </Modal>
    );
  }

  const submitAction = () => {
    if (busy || !email.trim()) {
      return undefined;
    }
    return submit;
  };

  return (
    <Modal title="Invite user" onClose={onClose} size="md" onSubmit={submitAction()}>
      <div className="mt-4 space-y-3">
        <Field label="Email">
          <Input
            type="email"
            value={email}
            autoComplete="off"
            placeholder="person@example.com"
            onChange={(e) => setEmail(e.target.value)}
          />
        </Field>
        <Field label="Display name" hint="Optional. They can change it themselves later.">
          <Input value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
        </Field>
        <Field label="Role">
          <Select value={role} onChange={(e) => setRole(e.target.value)}>
            <option value="basic">Basic</option>
            <option value="admin">Admin</option>
          </Select>
        </Field>
        <ErrorText>{error}</ErrorText>
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button disabled={busy || !email.trim()} onClick={submit}>
            {busy ? 'Inviting…' : 'Invite'}
          </Button>
        </div>
      </div>
    </Modal>
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
