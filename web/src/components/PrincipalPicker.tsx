import { useEffect, useState } from 'react';
import { api, type Principals, type VisibilityGrant } from '../api';

const selectCls =
  'rounded-button border border-hairline bg-surface-1 px-2 py-1.5 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-accent';

// PrincipalPicker grants users and groups access to something. It reads
// /api/principals so a non-admin creator can name people too.
export default function PrincipalPicker({
  grants,
  onAdd,
  onRemove,
}: {
  grants: VisibilityGrant[];
  onAdd: (type: 'user' | 'group', id: string) => void;
  onRemove: (type: 'user' | 'group', id: string) => void;
}) {
  const [principals, setPrincipals] = useState<Principals>({ users: [], groups: [] });

  useEffect(() => {
    api
      .get<Principals>('/api/principals')
      .then((p) => setPrincipals({ users: p?.users ?? [], groups: p?.groups ?? [] }))
      .catch(() => setPrincipals({ users: [], groups: [] }));
  }, []);

  const label = (g: VisibilityGrant) => {
    if (g.principal_type === 'user') {
      return principals.users.find((u) => u.id === g.principal_id)?.email ?? g.principal_id;
    }
    return `group: ${principals.groups.find((x) => x.id === g.principal_id)?.name ?? g.principal_id}`;
  };
  const granted = (type: 'user' | 'group', id: string) =>
    grants.some((g) => g.principal_type === type && g.principal_id === id);
  const add = (type: 'user' | 'group', id: string) => {
    if (id) {
      onAdd(type, id);
    }
  };

  return (
    <div>
      <div className="flex flex-wrap gap-2">
        {grants.length === 0 && <span className="text-xs text-muted">Only the creator and admins.</span>}
        {grants.map((g) => (
          <span
            key={`${g.principal_type}:${g.principal_id}`}
            className="inline-flex items-center gap-1.5 rounded-pill border border-hairline bg-surface-2 px-2 py-0.5 text-xs text-content"
          >
            {label(g)}
            <button
              type="button"
              className="text-muted hover:text-red-400"
              onClick={() => onRemove(g.principal_type, g.principal_id)}
              aria-label={`Remove ${label(g)}`}
            >
              ×
            </button>
          </span>
        ))}
      </div>
      <div className="mt-2 flex gap-2">
        <select aria-label="Add a user" className={selectCls} value="" onChange={(e) => add('user', e.target.value)}>
          <option value="">Add user…</option>
          {principals.users
            .filter((u) => !granted('user', u.id))
            .map((u) => (
              <option key={u.id} value={u.id}>
                {u.email}
              </option>
            ))}
        </select>
        <select aria-label="Add a group" className={selectCls} value="" onChange={(e) => add('group', e.target.value)}>
          <option value="">Add group…</option>
          {principals.groups
            .filter((g) => !granted('group', g.id))
            .map((g) => (
              <option key={g.id} value={g.id}>
                {g.name}
              </option>
            ))}
        </select>
      </div>
    </div>
  );
}
