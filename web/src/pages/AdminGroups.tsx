import { useEffect, useState } from 'react';
import ConfirmModal from '../components/ConfirmModal';
import { api, type Group, type GroupMember } from '../api';
import { useAuth } from '../auth';
import { Button, Card, ErrorText, Field, Input, Pill, Select } from '../components/ui';
import PageHeader from '../components/PageHeader';
import Avatar from '../components/Avatar';
import Modal from '../components/Modal';
import EmptyState from '../components/EmptyState';
import { matchesQuery } from '../lib/search';

export default function AdminGroups() {
  const { user: me } = useAuth();
  const isAdmin = me?.role === 'admin';
  const [groups, setGroups] = useState<Group[]>([]);
  const [creating, setCreating] = useState(false);
  const [search, setSearch] = useState('');
  const [error, setError] = useState('');

  // /api/groups is scoped to the caller: every group for an admin, the moderated
  // ones for anybody else. The page is the same either way.
  const load = () => api.get<Group[]>('/api/groups').then((g) => setGroups(g ?? []));
  useEffect(() => {
    load();
  }, []);

  const act = async (fn: () => Promise<unknown>) => {
    setError('');
    try {
      await fn();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed');
    }
    load();
  };

  // A moderator standing themselves down would leave a group they can no longer
  // manage, so their own row is theirs to read only. An admin can always fix it.
  const own = (m: GroupMember) => !isAdmin && m.user_id === me?.id;
  const setRole = (groupId: string, m: GroupMember, role: string) =>
    act(() => api.patch(`/api/groups/${groupId}/members/${m.user_id}`, { role }));
  const removeMember = (groupId: string, m: GroupMember) =>
    act(() => api.del(`/api/groups/${groupId}/members/${m.user_id}`));
  const [confirming, setConfirming] = useState<Group | null>(null);
  const remove = (groupId: string) => act(() => api.del(`/api/admin/groups/${groupId}`));

  // A group is searchable by its members' emails: you look for the person, not the label.
  const shown = groups.filter((g) =>
    matchesQuery(search, g.name, g.members.map((m) => `${m.email} ${m.display_name}`).join(' ')),
  );

  return (
    <div>
      <PageHeader
        title="User groups"
        search={{ value: search, onChange: setSearch, placeholder: 'Search groups…' }}
        action={isAdmin ? <Button onClick={() => setCreating(true)}>New group</Button> : undefined}
      />

      {error && (
        <div className="mt-4">
          <ErrorText>{error}</ErrorText>
        </div>
      )}

      <div className="mt-6 space-y-3">
        {groups.length === 0 && isAdmin && (
          <EmptyState
            title="No groups yet"
            description="Create a group to grant visibility and credential access to several users at once."
            action={<Button onClick={() => setCreating(true)}>New group</Button>}
          />
        )}
        {groups.length === 0 && !isAdmin && (
          <EmptyState
            title="No groups to manage"
            description="A group's moderator manages its members. An admin appoints one."
          />
        )}
        {groups.length > 0 && shown.length === 0 && (
          <p className="text-sm text-muted">No group matches the search.</p>
        )}
        {shown.map((g) => (
          <Card key={g.id} className="p-5">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-medium text-content">{g.name}</h2>
              {isAdmin && (
                <Button variant="danger" onClick={() => setConfirming(g)}>
                  Delete
                </Button>
              )}
            </div>

            <div className="mt-3 divide-y divide-hairline border-y border-hairline">
              {g.members.length === 0 && <p className="py-2 text-xs text-muted">No members</p>}
              {g.members.map((m) => (
                <div key={m.user_id} className="flex items-center justify-between gap-3 py-2">
                  <div className="flex min-w-0 items-center gap-2.5">
                    <Avatar url={m.avatar_url} name={m.display_name} email={m.email} size={28} />
                    <div className="min-w-0">
                      <p className="truncate text-sm text-content">{m.display_name || m.email}</p>
                      {m.display_name && <p className="truncate text-xs text-muted">{m.email}</p>}
                    </div>
                    {m.user_id === me?.id && <Pill>you</Pill>}
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <Select
                      value={m.role}
                      aria-label={`Role of ${m.email} in ${g.name}`}
                      disabled={own(m)}
                      onChange={(e) => setRole(g.id, m, e.target.value)}
                    >
                      <option value="member">Member</option>
                      <option value="moderator">Moderator</option>
                    </Select>
                    <Button variant="secondary" disabled={own(m)} onClick={() => removeMember(g.id, m)}>
                      Remove
                    </Button>
                  </div>
                </div>
              ))}
            </div>

            <AddMember group={g} onAdded={load} onError={setError} />
          </Card>
        ))}
      </div>

      {confirming && (
        <ConfirmModal
          title={`Delete ${confirming.name}?`}
          body="The group is deleted along with every visibility grant and credential access it carries. Its members keep their accounts."
          confirmLabel="Delete"
          onConfirm={() => remove(confirming.id)}
          onClose={() => setConfirming(null)}
        />
      )}

      {creating && (
        <CreateGroupModal
          onClose={() => setCreating(false)}
          onSaved={() => {
            setCreating(false);
            load();
          }}
        />
      )}
    </div>
  );
}

// Membership is added by address, not from a picker: a moderator is not handed the
// user list, and an admin already knows the address of whoever they are adding.
function AddMember({
  group,
  onAdded,
  onError,
}: {
  group: Group;
  onAdded: () => void;
  onError: (msg: string) => void;
}) {
  const [email, setEmail] = useState('');
  const [role, setRole] = useState('member');
  const [busy, setBusy] = useState(false);

  const add = async () => {
    setBusy(true);
    onError('');
    try {
      await api.post(`/api/groups/${group.id}/members`, { email, role });
      setEmail('');
      onAdded();
    } catch (e) {
      onError(e instanceof Error ? e.message : 'could not add that member');
    }
    setBusy(false);
  };

  return (
    <form
      className="mt-3 flex flex-wrap items-end gap-2"
      onSubmit={(e) => {
        e.preventDefault();
        if (!busy && email.trim()) {
          add();
        }
      }}
    >
      <div className="min-w-56 flex-1">
        <Field label="Add member by email">
          <Input
            type="email"
            value={email}
            placeholder="person@example.com"
            onChange={(e) => setEmail(e.target.value)}
          />
        </Field>
      </div>
      <Select value={role} aria-label="Role of the new member" onChange={(e) => setRole(e.target.value)}>
        <option value="member">Member</option>
        <option value="moderator">Moderator</option>
      </Select>
      <Button type="submit" variant="secondary" disabled={busy || !email.trim()}>
        {busy ? 'Adding…' : 'Add'}
      </Button>
    </form>
  );
}

function CreateGroupModal({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const create = async () => {
    setError('');
    setBusy(true);
    try {
      await api.post('/api/admin/groups', { name });
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed');
    } finally {
      setBusy(false);
    }
  };

  const submitAction = () => {
    if (busy || !name.trim()) {
      return undefined;
    }
    return create;
  };

  return (
    <Modal title="New group" onClose={onClose} size="md" onSubmit={submitAction()}>
      <div className="mt-4 space-y-3">
        <Field label="Name">
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="ops" />
        </Field>
        <ErrorText>{error}</ErrorText>
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={create} disabled={busy || !name.trim()}>
            {busy ? 'Creating…' : 'Create'}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
