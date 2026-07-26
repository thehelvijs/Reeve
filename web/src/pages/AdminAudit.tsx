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

export default function AdminAudit() {
  const [reveals, setReveals] = useState<RevealAuditEntry[]>([]);
  const [grants, setGrants] = useState<GrantAuditEntry[]>([]);
  const [users, setUsers] = useState<Record<string, string>>({});
  const [hosts, setHosts] = useState<Record<string, string>>({});

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

  return (
    <div>
      <PageHeader title="Audit log" subtitle="Who revealed what, and every grant or revoke." />

      <h2 className="mt-8 text-sm font-medium text-content">Credential reveals</h2>
      <Card className="mt-2 divide-y divide-hairline">
        {reveals.length === 0 && <p className="px-4 py-3 text-sm text-muted">No reveals yet.</p>}
        {reveals.map((r) => (
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
        {grants.length === 0 && <p className="px-4 py-3 text-sm text-muted">No grants yet.</p>}
        {grants.map((g) => (
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
