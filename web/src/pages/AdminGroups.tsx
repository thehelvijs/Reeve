import { useEffect, useState } from 'react';
import ConfirmModal from '../components/ConfirmModal';
import { api, type AdminUser, type Group } from '../api';
import { Button, Card, ErrorText, Field, Input } from '../components/ui';
import PageHeader from '../components/PageHeader';
import Modal from '../components/Modal';
import EmptyState from '../components/EmptyState';
import { matchesQuery } from '../lib/search';

export default function AdminGroups() {
  const [groups, setGroups] = useState<Group[]>([]);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [creating, setCreating] = useState(false);
  const [search, setSearch] = useState('');

  const load = () => {
    api.get<Group[]>('/api/admin/groups').then((g) => setGroups(g ?? []));
    api.get<AdminUser[]>('/api/admin/users').then((u) => setUsers(u ?? []));
  };
  useEffect(() => {
    load();
  }, []);

  const emailFor = (id: string) => users.find((u) => u.id === id)?.email ?? id;

  const addMember = async (groupId: string, userId: string) => {
    if (!userId) {
      return;
    }
    await api.put(`/api/admin/groups/${groupId}/members/${userId}`);
    load();
  };
  const removeMember = async (groupId: string, userId: string) => {
    await api.del(`/api/admin/groups/${groupId}/members/${userId}`);
    load();
  };
  const [confirming, setConfirming] = useState<Group | null>(null);

  const remove = async (groupId: string) => {
    await api.del(`/api/admin/groups/${groupId}`);
    load();
  };

  // A group is searchable by its members' emails: an admin looks for the person, not the label.
  const shown = groups.filter((g) => matchesQuery(search, g.name, g.members.map(emailFor).join(' ')));

  return (
    <div>
      <PageHeader
        title="User groups"
        subtitle="Assign visibility and access to a set of users at once. For grouping services, use Collections."
        search={{ value: search, onChange: setSearch, placeholder: 'Search groups…' }}
        action={<Button onClick={() => setCreating(true)}>New group</Button>}
      />

      <div className="mt-6 space-y-3">
        {groups.length === 0 && (
          <EmptyState
            title="No groups yet"
            description="Create a group to grant visibility and credential access to several users at once."
            action={<Button onClick={() => setCreating(true)}>New group</Button>}
          />
        )}
        {groups.length > 0 && shown.length === 0 && (
          <p className="text-sm text-muted">No group matches the search.</p>
        )}
        {shown.map((g) => (
          <Card key={g.id} className="p-5">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-medium text-content">{g.name}</h2>
              <Button variant="danger" onClick={() => setConfirming(g)}>
                Delete
              </Button>
            </div>
            <div className="mt-3 flex flex-wrap gap-2">
              {g.members.length === 0 && <span className="text-xs text-muted">No members</span>}
              {g.members.map((m) => (
                <span
                  key={m}
                  className="inline-flex items-center gap-1.5 rounded-pill border border-hairline bg-surface-2 px-2 py-0.5 text-xs text-content"
                >
                  {emailFor(m)}
                  <button
                    className="text-muted hover:text-down"
                    onClick={() => removeMember(g.id, m)}
                    aria-label={`Remove ${emailFor(m)}`}
                  >
                    ×
                  </button>
                </span>
              ))}
            </div>
            <div className="mt-3">
              <select
                className="rounded-button border border-hairline-strong bg-canvas px-2 py-1.5 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-link"
                value=""
                onChange={(e) => addMember(g.id, e.target.value)}
              >
                <option value="">Add member…</option>
                {users
                  .filter((u) => !g.members.includes(u.id))
                  .map((u) => (
                    <option key={u.id} value={u.id}>
                      {u.email}
                    </option>
                  ))}
              </select>
            </div>
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
