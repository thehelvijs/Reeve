import { Link } from 'react-router-dom';
import { api, type AccessRequest, type AlertEvent, type Host, type Tool, type ToolStatus } from '../api';
import { useAuth } from '../auth';
import { Button, Card, Pill } from '../components/ui';
import PageHeader from '../components/PageHeader';
import MetricBar from '../components/MetricBar';
import ServiceDots from '../components/ServiceDots';
import { Skeleton } from '../components/Skeleton';
import EmptyState from '../components/EmptyState';
import EntityIcon from '../components/EntityIcon';
import { fmtBytes } from '../lib/format';
import { useResource } from '../lib/cache';

export default function Dashboard() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const toolsRes = useResource<Tool[]>('/api/v1/tools', () => api.get<Tool[]>('/api/v1/tools'), 15000);
  const hostsRes = useResource<Host[]>('/api/v1/hosts', () => api.get<Host[]>('/api/v1/hosts'), 15000);
  const reqRes = useResource<AccessRequest[]>(
    '/api/v1/access-requests?box=inbox',
    () => api.get<AccessRequest[]>('/api/v1/access-requests?box=inbox'),
    15000,
  );
  const alertsRes = useResource<AlertEvent[]>(
    isAdmin ? '/api/v1/admin/alerts' : '',
    () => (isAdmin ? api.get<AlertEvent[]>('/api/v1/admin/alerts') : Promise.resolve([])),
    15000,
  );

  const tools = toolsRes.data ?? [];
  const hosts = hostsRes.data ?? [];
  const reviewCount = (reqRes.data ?? []).length;
  const alertCount = (alertsRes.data ?? []).filter((e) => !e.resolved_at).length;
  const loading = toolsRes.loading || hostsRes.loading;

  const count = (s: ToolStatus) => tools.filter((t) => t.status === s).length;
  const hostsOnline = hosts.filter((h) => h.status === 'online').length;
  const toolsFor = (id: string) => tools.filter((t) => t.host_id === id);
  const unassigned = tools.filter((t) => !t.host_id);

  return (
    <div>
      <PageHeader title="Dashboard" subtitle="Your whole infrastructure at a glance." />

      {loading && (
        <>
          <div className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-[76px]" />
            ))}
          </div>
          <Skeleton className="mt-8 h-4 w-16" />
          <div className="mt-2 grid grid-cols-1 gap-3 lg:grid-cols-2">
            <Skeleton className="h-40" />
            <Skeleton className="h-40" />
          </div>
        </>
      )}

      {!loading && (
        <>
          <div className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
            <Stat label="Up" value={count('up')} tone="up" />
            <Stat label="Down" value={count('down') + count('agent_offline')} tone="down" />
            <Stat label="Unknown" value={count('unknown')} tone="muted" />
            <Stat label="Hosts online" value={`${hostsOnline}/${hosts.length}`} tone="muted" />
            <Link to="/requests">
              <Stat label="To review" value={reviewCount} tone={reviewCount > 0 ? 'accent' : 'muted'} />
            </Link>
            {isAdmin && (
              <Link to="/admin/alerts">
                <Stat label="Alerts firing" value={alertCount} tone={alertCount > 0 ? 'down' : 'muted'} />
              </Link>
            )}
          </div>

          <h2 className="mt-8 text-sm font-medium text-content">Hosts</h2>
          {hosts.length === 0 ? (
            <div className="mt-2">
              <EmptyState
                title="No hosts yet"
                description="Enroll a machine from the Hosts page to see its status and metrics here."
                action={
                  <Link to="/hosts">
                    <Button variant="secondary">Go to Hosts</Button>
                  </Link>
                }
              />
            </div>
          ) : (
            <div className="mt-2 grid grid-cols-1 gap-3 lg:grid-cols-2">
              {hosts.map((h) => (
                <HostCard key={h.id} host={h} tools={toolsFor(h.id)} />
              ))}
            </div>
          )}
        </>
      )}

      {!loading && unassigned.length > 0 && (
        <>
          <h2 className="mt-8 text-sm font-medium text-content">Unassigned services</h2>
          <Card className="mt-2 p-4">
            <ServiceDots tools={unassigned} />
          </Card>
        </>
      )}
    </div>
  );
}

function HostCard({ host, tools }: { host: Host; tools: Tool[] }) {
  const m = host.metrics;
  const pct = (used: number, total: number) => {
    if (total > 0) {
      return (used / total) * 100;
    }
    return 0;
  };
  let tone: 'up' | 'down' | 'muted' = 'muted';
  if (host.status === 'online') {
    tone = 'up';
  } else if (host.status === 'offline') {
    tone = 'down';
  }
  return (
    <Link to={`/hosts/${host.id}`}>
      <Card className="p-4 transition-colors hover:bg-surface-2">
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2">
            <EntityIcon url={host.icon_url} name={host.name} size={28} />
            <HostDot status={host.status} />
            <span className="truncate text-sm font-medium text-content">{host.name}</span>
            <span className="truncate text-xs text-muted">{host.os}</span>
          </div>
          <Pill tone={tone}>{host.status}</Pill>
        </div>

        {m ? (
          <div className="mt-3 space-y-2">
            <MetricBar label="CPU" pct={m.cpu_pct} />
            <MetricBar
              label="Memory"
              pct={pct(m.mem_used, m.mem_total)}
              detail={`${fmtBytes(m.mem_used)} / ${fmtBytes(m.mem_total)}`}
            />
            <MetricBar
              label="Disk"
              pct={pct(m.disk_used, m.disk_total)}
              detail={`${fmtBytes(m.disk_used)} / ${fmtBytes(m.disk_total)}`}
            />
          </div>
        ) : (
          <p className="mt-3 text-xs text-muted">No metrics yet.</p>
        )}

        <div className="mt-3 border-t border-hairline pt-3">
          <ServiceDots tools={tools} />
        </div>
      </Card>
    </Link>
  );
}

function HostDot({ status }: { status: Host['status'] }) {
  let color = 'bg-muted';
  if (status === 'online') {
    color = 'bg-green-400';
  } else if (status === 'offline') {
    color = 'bg-red-400';
  }
  return <span className={`inline-block h-2 w-2 shrink-0 rounded-full ${color}`} />;
}

function Stat({
  label,
  value,
  tone,
}: {
  label: string;
  value: number | string;
  tone: 'up' | 'down' | 'muted' | 'accent';
}) {
  const colors = {
    up: 'text-green-400',
    down: 'text-red-400',
    accent: 'text-accent',
    muted: 'text-content',
  };
  const color = colors[tone];
  return (
    <Card className="p-4 transition-colors hover:bg-surface-2">
      <p className="text-xs text-muted">{label}</p>
      <p className={`mt-1 text-3xl font-semibold tracking-tight ${color}`}>{value}</p>
    </Card>
  );
}
