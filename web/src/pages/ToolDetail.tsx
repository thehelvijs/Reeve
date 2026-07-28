import { useCallback, useEffect, useState } from 'react';
import ConfirmModal from '../components/ConfirmModal';
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
import { Button, Card, Facts, Pill, Section, Select } from '../components/ui';
import PageHeader from '../components/PageHeader';
import StatusPill from '../components/StatusPill';
import EntityIcon from '../components/EntityIcon';
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
      .get<Tool>(`/api/tools/${id}`)
      .then(setTool)
      .catch(() => setNotFound(true));
  }, [id]);
  useEffect(() => {
    load();
  }, [load]);

  const [confirming, setConfirming] = useState(false);

  const remove = async () => {
    if (!tool) {
      return;
    }
    await api.del(`/api/tools/${tool.id}`);
    navigate('/services');
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
      <PageHeader
        title={tool.name}
        icon={<EntityIcon url={tool.icon_url} name={tool.name} size={36} />}
        badges={
          <>
            <StatusPill status={tool.status} />
            {tool.visibility === 'restricted' && <Pill tone="down">restricted</Pill>}
          </>
        }
        subtitle={tool.description || undefined}
        meta={<UptimeSummary path={`/api/tools/${tool.id}/uptime`} />}
        action={
          tool.can_edit && (
            <>
              <Button variant="secondary" onClick={() => navigate(`/services/${tool.id}/edit`)}>
                Edit
              </Button>
              <Button variant="danger" onClick={() => setConfirming(true)}>
                Delete
              </Button>
            </>
          )
        }
      />

      {confirming && (
        <ConfirmModal
          title={`Delete ${tool.name}?`}
          body="The service is removed from the catalog along with its collections, visibility grants and uptime history. The machine it points at is untouched."
          confirmLabel="Delete"
          onConfirm={remove}
          onClose={() => setConfirming(false)}
        />
      )}

      <div className="mt-6 space-y-6">
        {tool.thumbnail_url && (
          <img
            src={tool.thumbnail_url}
            alt=""
            className="w-full max-w-lg rounded-card border border-hairline object-cover"
          />
        )}

        <Section title="Address" description="Where this service answers.">
          <Card className="p-4">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="min-w-0">
                <p className="text-eyebrow font-semibold uppercase text-muted">Endpoint</p>
                <code className="font-mono text-sm text-content">{endpointString(tool) || '—'}</code>
                <p className="mt-3 text-eyebrow font-semibold uppercase text-muted">Share this instead</p>
                <code className="font-mono text-xs text-content">{goURL(tool)}</code>
                <p className="mt-1 text-xs text-muted">
                  It resolves at click time, so it keeps working when the host&rsquo;s address changes.
                </p>
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
        </Section>

        <Section title="Details">
          <Card className="p-4">
            <Facts
              items={[
                {
                  label: 'Collections',
                  value:
                    tool.collections.length === 0 ? (
                      '—'
                    ) : (
                      <span className="flex flex-wrap gap-1.5">
                        {tool.collections.map((c) => (
                          <Link key={c.id} to={`/collections/${c.id}`}>
                            <Pill>{c.name}</Pill>
                          </Link>
                        ))}
                      </span>
                    ),
                },
                {
                  label: 'Source',
                  value: `${tool.source_type}${tool.source_ref ? ` · ${tool.source_ref}` : ''}`,
                },
                { label: 'Tags', value: tool.tags.join(', ') || '—' },
              ]}
            />
          </Card>
        </Section>

        {tool.host_id && (
          <Section title="Credentials">
            <Card className="p-4">
              <p className="text-xs text-muted">
                Logins live on the machine, not the service.{' '}
                <Link
                  to={`/hosts/${tool.host_id}`}
                  className="text-link underline underline-offset-2 hover:text-link-hover"
                >
                  View this host
                </Link>{' '}
                to see or request them.
              </p>
            </Card>
          </Section>
        )}

        <EventHistory path={`/api/tools/${tool.id}/events`} />

        {user?.role === 'admin' && tool.visibility === 'restricted' && (
          <VisibilityManager toolId={tool.id} />
        )}
      </div>
    </div>
  );
}

function VisibilityManager({ toolId }: { toolId: string }) {
  const [grants, setGrants] = useState<VisibilityGrant[]>([]);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);

  const load = useCallback(() => {
    api.get<VisibilityGrant[]>(`/api/tools/${toolId}/visibility`).then((g) => setGrants(g ?? []));
    api.get<AdminUser[]>('/api/admin/users').then((u) => setUsers(u ?? []));
    api.get<Group[]>('/api/admin/groups').then((g) => setGroups(g ?? []));
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
    await api.put(`/api/tools/${toolId}/visibility/${ptype}/${pid}`);
    load();
  };
  const remove = async (g: VisibilityGrant) => {
    await api.del(`/api/tools/${toolId}/visibility/${g.principal_type}/${g.principal_id}`);
    load();
  };

  return (
    <Section title="Who can see this service" description="Everyone else gets a 404, admins excepted.">
      <Card className="p-4">
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
                onClick={() => remove(g)}
                aria-label={`Remove ${label(g)}`}
                className="rounded-pill text-muted transition-colors hover:text-down focus:outline-none focus-visible:ring-2 focus-visible:ring-link"
              >
                ×
              </button>
            </span>
          ))}
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <Select value="" aria-label="Grant a user access" onChange={(e) => add('user', e.target.value)}>
            <option value="">Add user…</option>
            {users.map((u) => (
              <option key={u.id} value={u.id}>
                {u.email}
              </option>
            ))}
          </Select>
          <Select value="" aria-label="Grant a group access" onChange={(e) => add('group', e.target.value)}>
            <option value="">Add group…</option>
            {groups.map((g) => (
              <option key={g.id} value={g.id}>
                {g.name}
              </option>
            ))}
          </Select>
        </div>
      </Card>
    </Section>
  );
}

