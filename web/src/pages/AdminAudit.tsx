import { useEffect, useState } from 'react';
import {
  api,
  type AdminUser,
  type GrantAuditEntry,
  type Host,
  type RevealAuditEntry,
} from '../api';
import { Card, Pill } from '../components/ui';
import PageHeader from '../components/PageHeader';
import { matchesQuery } from '../lib/search';

export default function AdminAudit() {
  const [reveals, setReveals] = useState<RevealAuditEntry[]>([]);
  const [grants, setGrants] = useState<GrantAuditEntry[]>([]);
  const [users, setUsers] = useState<Record<string, string>>({});
  const [hosts, setHosts] = useState<Record<string, string>>({});
  const [search, setSearch] = useState('');

  useEffect(() => {
    api.get<RevealAuditEntry[]>('/api/admin/audit/reveals').then((r) => setReveals(r ?? []));
    api.get<GrantAuditEntry[]>('/api/admin/audit/grants').then((g) => setGrants(g ?? []));
    api.get<AdminUser[]>('/api/admin/users').then((us) => {
      const m: Record<string, string> = {};
      (us ?? []).forEach((u) => (m[u.id] = u.email));
      setUsers(m);
    });
    api.get<Host[]>('/api/hosts').then((hs) => {
      const m: Record<string, string> = {};
      (hs ?? []).forEach((h) => (m[h.id] = h.name));
      setHosts(m);
    });
  }, []);

  const u = (id: string) => users[id] ?? id;
  const hn = (id: string) => hosts[id] ?? id;

  // One search covers both lists: an investigation starts from a name, a host or
  // an address, not from which of the two tables recorded it.
  const shownReveals = reveals.filter((r) => matchesQuery(search, u(r.user_id), hn(r.host_id), r.source_ip));
  const shownGrants = grants.filter((g) =>
    matchesQuery(search, u(g.actor_id), hn(g.host_id), g.action, g.principal_type, g.principal_id),
  );

  let noReveals = 'No reveals yet.';
  let noGrants = 'No grants yet.';
  if (search.trim()) {
    noReveals = 'No reveal matches the search.';
    noGrants = 'No grant matches the search.';
  }

  return (
    <div>
      <PageHeader
        title="Audit log"
        search={{ value: search, onChange: setSearch, placeholder: 'Search audit log…' }}
      />

      <h2 className="mt-8 text-sm font-medium text-content">Credential reveals</h2>
      <Card className="mt-2 divide-y divide-hairline">
        {shownReveals.length === 0 && <p className="px-4 py-3 text-sm text-muted">{noReveals}</p>}
        {shownReveals.map((r) => (
          <div key={r.id} className="flex items-center justify-between px-4 py-2.5 text-sm">
            <span className="text-content">{u(r.user_id)}</span>
            <span className="text-muted">revealed on {hn(r.host_id)}</span>
            <span className="font-mono text-xs text-muted">{r.source_ip}</span>
            <span className="text-xs text-muted">{new Date(r.revealed_at).toLocaleString()}</span>
          </div>
        ))}
      </Card>

      <h2 className="mt-6 text-sm font-medium text-content">Grants &amp; revokes</h2>
      <Card className="mt-2 divide-y divide-hairline">
        {shownGrants.length === 0 && <p className="px-4 py-3 text-sm text-muted">{noGrants}</p>}
        {shownGrants.map((g) => (
          <div key={g.id} className="flex items-center justify-between px-4 py-2.5 text-sm">
            <span className="text-content">{u(g.actor_id)}</span>
            <Pill tone={g.action === 'grant' ? 'up' : 'down'}>{g.action}</Pill>
            <span className="text-muted">
              {g.principal_type}:{g.principal_type === 'user' ? u(g.principal_id) : g.principal_id} on {hn(g.host_id)}
            </span>
            <span className="text-xs text-muted">{new Date(g.at).toLocaleString()}</span>
          </div>
        ))}
      </Card>
    </div>
  );
}
