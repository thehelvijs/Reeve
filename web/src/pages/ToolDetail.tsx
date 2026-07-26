import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  api,
  endpointString,
  goURL,
  type AdminUser,
  type Group,
  type Tool,
  type VisibilityGrant,
} from '../api';
import { useAuth } from '../auth';
import { Button, Card, Pill } from '../components/ui';
import BackLink from '../components/BackLink';
import StatusPill from '../components/StatusPill';
import EntityIcon from '../components/EntityIcon';
import CredentialsSection from '../components/CredentialsSection';
import EventHistory from '../components/EventHistory';
import UptimeSummary from '../components/UptimeSummary';

export default function ToolDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const [tool, setTool] = useState<Tool | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [copied, setCopied] = useState(false);

  const load = useCallback(() => {
    api
      .get<Tool>(`/api/v1/tools/${id}`)
      .then(setTool)
      .catch(() => setNotFound(true));
  }, [id]);
  useEffect(() => {
    load();
  }, [load]);

  const remove = async () => {
    if (!tool || !confirm(`Delete "${tool.name}"?`)) {
      return;
    }
    await api.del(`/api/v1/tools/${tool.id}`);
    navigate('/catalog');
  };

  const copy = async () => {
    if (!tool) {
      return;
    }
    await navigator.clipboard.writeText(new URL(goURL(tool), window.location.origin).toString());
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  if (notFound) {
    return <p className="text-sm text-muted">Tool not found.</p>;
  }
  if (!tool) {
    return <p className="text-sm text-muted">Loading…</p>;
  }

  return (
    <div>
      <BackLink to="/catalog">Services</BackLink>

      <div className="mt-3 flex items-start justify-between">
        <div className="flex min-w-0 items-start gap-3">
          <EntityIcon url={tool.icon_url} name={tool.name} size={40} />
          <div className="min-w-0">
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-semibold tracking-tight text-content">{tool.name}</h1>
              <StatusPill status={tool.status} />
              {tool.visibility === 'restricted' && <Pill tone="down">restricted</Pill>}
            </div>
            {tool.description && <p className="mt-1 text-sm text-muted">{tool.description}</p>}
            <UptimeSummary path={`/api/v1/tools/${tool.id}/uptime`} />
          </div>
        </div>
        {tool.can_edit && (
          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => navigate(`/catalog/${tool.id}/edit`)}>
              Edit
            </Button>
            <Button variant="danger" onClick={remove}>
              Delete
            </Button>
          </div>
        )}
      </div>

      {tool.thumbnail_url && (
        <img
          src={tool.thumbnail_url}
          alt=""
          className="mt-6 w-full max-w-lg rounded-card border border-hairline object-cover"
        />
      )}

      <Card className="mt-6 p-5">
        <div className="flex items-center justify-between gap-4">
          <div className="min-w-0">
            <p className="text-xs text-muted">Endpoint</p>
            <code className="font-mono text-sm text-content">{endpointString(tool) || '—'}</code>
            <p className="mt-2 text-xs text-muted">
              Share this link instead — it resolves at click time, so it keeps working when the
              host&rsquo;s address changes:
            </p>
            <code className="font-mono text-xs text-content">{goURL(tool)}</code>
          </div>
          <div className="flex shrink-0 gap-2">
            <Button variant="secondary" onClick={copy}>
              {copied ? 'Copied' : 'Copy link'}
            </Button>
            <a href={goURL(tool)} target="_blank" rel="noreferrer">
              <Button>Open</Button>
            </a>
          </div>
        </div>
      </Card>

      <div className="mt-4 grid grid-cols-2 gap-4">
        <div>
          <p className="text-xs text-muted">Collections</p>
          {tool.collections.length === 0 && <p className="text-sm text-content">—</p>}
          {tool.collections.length > 0 && (
            <div className="mt-1 flex flex-wrap gap-1.5">
              {tool.collections.map((c) => (
                <Link key={c.id} to={`/collections/${c.id}`}>
                  <Pill>{c.name}</Pill>
                </Link>
              ))}
            </div>
          )}
        </div>
        <Detail label="Source" value={`${tool.source_type}${tool.source_ref ? ` · ${tool.source_ref}` : ''}`} />
        <Detail label="Tags" value={tool.tags.join(', ')} />
      </div>

      <CredentialsSection tool={tool} />

      <EventHistory path={`/api/v1/tools/${tool.id}/events`} />

      {user?.role === 'admin' && tool.visibility === 'restricted' && (
        <VisibilityManager toolId={tool.id} />
      )}
    </div>
  );
}

function Detail({ label, value }: { label: string; value?: string }) {
  return (
    <div>
      <p className="text-xs text-muted">{label}</p>
      <p className="text-sm text-content">{value || '—'}</p>
    </div>
  );
}

function VisibilityManager({ toolId }: { toolId: string }) {
  const [grants, setGrants] = useState<VisibilityGrant[]>([]);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);

  const load = useCallback(() => {
    api.get<VisibilityGrant[]>(`/api/v1/tools/${toolId}/visibility`).then((g) => setGrants(g ?? []));
    api.get<AdminUser[]>('/api/v1/admin/users').then((u) => setUsers(u ?? []));
    api.get<Group[]>('/api/v1/admin/groups').then((g) => setGroups(g ?? []));
  }, [toolId]);
  useEffect(() => {
    load();
  }, [load]);

  const label = (g: VisibilityGrant) => {
    if (g.principal_type === 'user') {
      return users.find((u) => u.id === g.principal_id)?.email ?? g.principal_id;
    }
    return `group: ${groups.find((x) => x.id === g.principal_id)?.name ?? g.principal_id}`;
  };
  const add = async (ptype: string, pid: string) => {
    if (!pid) {
      return;
    }
    await api.put(`/api/v1/tools/${toolId}/visibility/${ptype}/${pid}`);
    load();
  };
  const remove = async (g: VisibilityGrant) => {
    await api.del(`/api/v1/tools/${toolId}/visibility/${g.principal_type}/${g.principal_id}`);
    load();
  };

  return (
    <Card className="mt-6 p-5">
      <p className="text-sm font-medium text-content">Who can see this tool</p>
      <div className="mt-3 flex flex-wrap gap-2">
        {grants.length === 0 && <span className="text-xs text-muted">Only the creator and admins.</span>}
        {grants.map((g) => (
          <span
            key={`${g.principal_type}:${g.principal_id}`}
            className="inline-flex items-center gap-1.5 rounded-pill border border-hairline bg-surface-2 px-2 py-0.5 text-xs text-content"
          >
            {label(g)}
            <button className="text-muted hover:text-red-400" onClick={() => remove(g)} aria-label="Remove">
              ×
            </button>
          </span>
        ))}
      </div>
      <div className="mt-3 flex gap-2">
        <select className={selectCls} value="" onChange={(e) => add('user', e.target.value)}>
          <option value="">Add user…</option>
          {users.map((u) => (
            <option key={u.id} value={u.id}>
              {u.email}
            </option>
          ))}
        </select>
        <select className={selectCls} value="" onChange={(e) => add('group', e.target.value)}>
          <option value="">Add group…</option>
          {groups.map((g) => (
            <option key={g.id} value={g.id}>
              {g.name}
            </option>
          ))}
        </select>
      </div>
    </Card>
  );
}

const selectCls =
  'rounded-button border border-hairline bg-surface-1 px-2 py-1.5 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-accent';

