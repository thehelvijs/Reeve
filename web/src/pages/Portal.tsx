import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, endpointString, type Host, type Tool } from '../api';
import { useAuth } from '../auth';
import { Button, Card, Pill } from '../components/ui';
import StatusPill from '../components/StatusPill';
import SearchBar from '../components/SearchBar';
import EntityIcon from '../components/EntityIcon';
import LeafletMap from '../components/LeafletMap';
import PortalGraph from '../components/PortalGraph';
import HostInfoModal from '../components/HostInfoModal';
import ToolInfoModal from '../components/ToolInfoModal';
import { groupToolsByHost } from '../lib/group';
import { SOURCE_URL, UI_VERSION } from '../version';

type View = 'list' | 'map' | 'graph';

export default function Portal() {
  const { user, loading } = useAuth();
  const navigate = useNavigate();
  const [tools, setTools] = useState<Tool[]>([]);
  const [hosts, setHosts] = useState<Host[]>([]);
  const [search, setSearch] = useState('');
  const [view, setView] = useState<View>('list');
  const [selected, setSelected] = useState<{ kind: 'host'; host: Host } | { kind: 'tool'; tool: Tool } | null>(null);

  useEffect(() => {
    if (loading) {
      return;
    }
    const toolsPath = user ? '/api/v1/tools' : '/api/v1/public/tools';
    const hostsPath = user ? '/api/v1/hosts' : '/api/v1/public/hosts';
    const load = () => {
      api.get<Tool[]>(toolsPath).then((t) => setTools(t ?? [])).catch(() => setTools([]));
      api.get<Host[]>(hostsPath).then((h) => setHosts(h ?? [])).catch(() => setHosts([]));
    };
    load();
    const poll = setInterval(load, 15000);
    return () => clearInterval(poll);
  }, [user, loading]);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) {
      return tools;
    }
    return tools.filter((t) => t.name.toLowerCase().includes(q) || t.description.toLowerCase().includes(q));
  }, [tools, search]);

  const groups = useMemo(() => groupToolsByHost(filtered, hosts), [filtered, hosts]);

  const openTool = (id: string) => {
    const t = tools.find((x) => x.id === id);
    if (t) {
      setSelected({ kind: 'tool', tool: t });
    }
  };
  const openHost = (id: string) => {
    const h = hosts.find((x) => x.id === id);
    if (h) {
      setSelected({ kind: 'host', host: h });
    }
  };

  const mapPoints = useMemo(
    () =>
      hosts
        .filter((h) => h.latitude != null && h.longitude != null)
        .map((h) => ({ id: h.id, lat: h.latitude as number, lon: h.longitude as number, label: h.name })),
    [hosts],
  );

  return (
    <div className="flex h-screen flex-col bg-canvas">
      <header className="relative flex shrink-0 items-center gap-4 border-b border-hairline bg-surface-1 px-8 py-3">
        <span className="flex shrink-0 items-baseline gap-1.5">
          <span className="text-sm font-medium tracking-tight text-accent">Reeve</span>
          <span className="text-[10px] font-medium text-muted">{UI_VERSION}</span>
          <a
            href={SOURCE_URL}
            target="_blank"
            rel="noreferrer"
            className="text-[10px] font-medium text-muted transition-colors hover:text-content"
          >
            source
          </a>
        </span>
        <div className="pointer-events-none absolute inset-x-0 hidden justify-center lg:flex">
          <div className="pointer-events-auto w-full max-w-sm">
            <SearchBar value={search} onChange={setSearch} />
          </div>
        </div>
        <Button className="ml-auto shrink-0" onClick={() => navigate(user ? '/dashboard' : '/login')}>
          Dashboard
        </Button>
      </header>

      <main className="flex min-h-0 flex-1 flex-col px-8 py-6">
        <div className="flex min-h-0 w-full flex-1 flex-col">
        <div className="mb-4 flex shrink-0 justify-center">
          <div className="inline-flex rounded-button border border-hairline p-0.5">
            {(['list', 'map', 'graph'] as View[]).map((v) => (
              <button
                key={v}
                type="button"
                onClick={() => setView(v)}
                className={`rounded-[5px] px-3 py-1 text-sm capitalize transition-colors ${
                  view === v ? 'bg-surface-2 text-content' : 'text-muted hover:text-content'
                }`}
              >
                {v}
              </button>
            ))}
          </div>
        </div>

        {view === 'map' && (
          <div className="relative min-h-0 flex-1">
            <LeafletMap points={mapPoints} onPointClick={openHost} height="100%" />
            {mapPoints.length === 0 && (
              <p className="absolute left-3 top-3 rounded-button bg-surface-3/90 px-2 py-1 text-xs text-muted">
                No hosts have a location set yet.
              </p>
            )}
          </div>
        )}

        {view === 'graph' && (
          <div className="min-h-0 flex-1">
            <PortalGraph tools={filtered} hosts={hosts} onOpenTool={openTool} onOpenHost={openHost} />
          </div>
        )}

        {view === 'list' && (
          <div className="min-h-0 flex-1 overflow-auto">
            {groups.length === 0 && <p className="text-sm text-muted">No tools to show.</p>}
            {groups.map((g) => (
          <section key={g.host?.id ?? 'unassigned'} className="mb-8">
            {g.host ? (
              <button
                type="button"
                onClick={() => openHost(g.host!.id)}
                className="flex items-center gap-2 rounded-button px-1 py-0.5 transition-colors hover:bg-surface-2"
              >
                <EntityIcon url={g.host.icon_url} name={g.host.name} size={22} />
                <HostDot status={g.host.status} />
                <h2 className="text-sm font-medium text-content">{g.host.name}</h2>
                <span className="text-xs text-muted">{g.tools.length}</span>
              </button>
            ) : (
              <div className="flex items-center gap-2 px-1">
                <HostDot status={undefined} />
                <h2 className="text-sm font-medium text-content">Unassigned</h2>
                <span className="text-xs text-muted">{g.tools.length}</span>
              </div>
            )}
            <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {g.tools.map((t) => (
                <button key={t.id} type="button" onClick={() => openTool(t.id)} className="text-left">
                  <Card className="flex h-full flex-col px-4 py-3 transition-colors hover:bg-surface-2">
                    <div className="flex items-start justify-between gap-2">
                      <EntityIcon url={t.icon_url} name={t.name} size={36} />
                      <StatusPill status={t.status} />
                    </div>
                    <p className="mt-2 truncate text-sm font-medium text-content">{t.name}</p>
                    <p className="truncate font-mono text-xs text-muted">{endpointString(t) || '—'}</p>
                    {t.category && (
                      <div className="mt-2 flex flex-wrap gap-1.5">
                        <Pill>{t.category}</Pill>
                      </div>
                    )}
                  </Card>
                </button>
              ))}
            </div>
          </section>
            ))}
          </div>
        )}
        </div>
      </main>

      {selected?.kind === 'host' && (
        <HostInfoModal
          host={selected.host}
          tools={tools.filter((t) => t.host_id === selected.host.id)}
          onClose={() => setSelected(null)}
          onOpenTool={openTool}
        />
      )}
      {selected?.kind === 'tool' && (
        <ToolInfoModal
          tool={selected.tool}
          hostName={hosts.find((h) => h.id === selected.tool.host_id)?.name}
          canOpen={Boolean(user)}
          onOpen={() => {
            navigate(`/catalog/${selected.tool.id}`);
          }}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  );
}

function HostDot({ status }: { status?: Host['status'] }) {
  let color = 'bg-muted';
  if (status === 'online') {
    color = 'bg-green-400';
  } else if (status === 'offline') {
    color = 'bg-red-400';
  }
  return <span className={`inline-block h-2 w-2 rounded-full ${color}`} />;
}
