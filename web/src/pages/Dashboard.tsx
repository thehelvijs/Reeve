import { Link } from 'react-router-dom';
import { api, type AccessRequest, type AlertEvent, type Host, type Tool, type ToolStatus } from '../api';
import { useAuth } from '../auth';
import { Button, Card, Eyebrow, Pill } from '../components/ui';
import PageHeader from '../components/PageHeader';
import MetricBar from '../components/MetricBar';
import ServiceDots from '../components/ServiceDots';
import { Skeleton } from '../components/Skeleton';
import EmptyState from '../components/EmptyState';
import EntityIcon from '../components/EntityIcon';
import { fmtBytes } from '../lib/format';
import { useResource } from '../lib/cache';
import { dotClass, hostTone } from '../lib/statusTone';

export default function Dashboard() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const toolsRes = useResource<Tool[]>('/api/tools', () => api.get<Tool[]>('/api/tools'), 15000);
  const hostsRes = useResource<Host[]>('/api/hosts', () => api.get<Host[]>('/api/hosts'), 15000);
  const reqRes = useResource<AccessRequest[]>(
    '/api/access-requests?box=inbox',
    () => api.get<AccessRequest[]>('/api/access-requests?box=inbox'),
    15000,
  );
  const alertsRes = useResource<AlertEvent[]>(
    isAdmin ? '/api/admin/alerts' : '',
    () => (isAdmin ? api.get<AlertEvent[]>('/api/admin/alerts') : Promise.resolve([])),
    15000,
  );

  const tools = toolsRes.data ?? [];
  const hosts = hostsRes.data ?? [];
  const reviewCount = (reqRes.data ?? []).length;
  const alertCount = (alertsRes.data ?? []).filter((e) => !e.resolved_at).length;
  const loading = toolsRes.loading || hostsRes.loading;

  const count = (s: ToolStatus) => tools.filter((t) => t.status === s).length;
  const total = tools.length;
  const hostsOnline = hosts.filter((h) => h.status === 'online').length;
  // A card whose count is zero is telling you nothing is wrong, which is what
  // the whole section already says by being absent.
  const hostsQuiet = hosts.length > 0 && hostsOnline < hosts.length;
  const attention = hostsQuiet || reviewCount > 0 || alertCount > 0;
  const toolsFor = (id: string) => tools.filter((t) => t.host_id === id);
  const unassigned = tools.filter((t) => !t.host_id);

  return (
    <div>
      <PageHeader title="Dashboard" />

      {loading && (
        <>
          <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-[92px]" />
            ))}
          </div>
          <div className="mt-6 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-[92px]" />
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
          {total > 0 && (
            <>
              <Eyebrow>Services</Eyebrow>
              <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
                <Tile
                  label="Up"
                  value={`${count('up')} of ${total}`}
                  meaning="Running as expected."
                  tone="up"
                />
                {count('down') > 0 && (
                  <Tile
                    label="Down"
                    value={count('down')}
                    meaning="The agent reports it stopped."
                    tone="down"
                  />
                )}
                {count('agent_offline') > 0 && (
                  <Tile
                    label="Unreachable"
                    value={count('agent_offline')}
                    meaning="Host offline, so its last state is unknown."
                    tone="warn"
                  />
                )}
                {count('unknown') > 0 && (
                  <Tile
                    label="Not monitored"
                    value={count('unknown')}
                    meaning="No agent source to check."
                    tone="muted"
                  />
                )}
              </div>
            </>
          )}

          {attention && (
            <>
              <div className="mt-6">
                <Eyebrow>Needs attention</Eyebrow>
              </div>
              <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {hostsQuiet && (
                  <Tile
                    label="Hosts online"
                    value={`${hostsOnline} of ${hosts.length}`}
                    meaning="The rest are not sending telemetry."
                    tone="warn"
                    to="/hosts"
                  />
                )}
                {reviewCount > 0 && (
                  <Tile
                    label="To review"
                    value={reviewCount}
                    meaning="Access requests waiting on you."
                    tone="link"
                    to="/requests"
                  />
                )}
                {isAdmin && alertCount > 0 && (
                  <Tile
                    label="Alerts firing"
                    value={alertCount}
                    meaning="Unresolved since they fired."
                    tone="down"
                    to="/admin/alerts"
                  />
                )}
              </div>
            </>
          )}

          <h2 className="mt-8 text-sm font-semibold text-content">Hosts</h2>
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
          <h2 className="mt-8 text-sm font-semibold text-content">Unassigned services</h2>
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
  return (
    <Link to={`/hosts/${host.id}`}>
      <Card className="p-4 transition-colors hover:bg-surface-1">
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2">
            <EntityIcon url={host.icon_url} name={host.name} size={28} />
            <HostDot status={host.status} />
            <span className="truncate text-sm font-medium text-content">{host.name}</span>
            <span className="truncate text-xs text-muted">{host.os}</span>
          </div>
          <Pill tone={hostTone(host.status)}>{host.status}</Pill>
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
  return <span className={`inline-block h-2 w-2 shrink-0 rounded-full ${dotClass(hostTone(status))}`} />;
}

// Every tile carries the number, what it is out of, and what the state means.
function Tile({
  label,
  value,
  meaning,
  tone,
  to,
}: {
  label: string;
  value: number | string;
  meaning: string;
  tone: 'up' | 'down' | 'warn' | 'muted' | 'link';
  to?: string;
}) {
  const colors = {
    up: 'text-up',
    down: 'text-down',
    warn: 'text-warn',
    link: 'text-link',
    muted: 'text-content',
  };
  const body = (
    <>
      <p className="text-eyebrow font-semibold uppercase text-muted">{label}</p>
      <p className={`mt-1 text-2xl font-semibold tabular-nums ${colors[tone]}`}>{value}</p>
      <p className="mt-1 text-xs text-muted">{meaning}</p>
    </>
  );
  if (to) {
    return (
      <Link
        to={to}
        className="rounded-card border border-hairline bg-canvas p-4 transition-colors hover:border-hairline-strong hover:bg-surface-1"
      >
        {body}
      </Link>
    );
  }
  return <Card className="p-4">{body}</Card>;
}
