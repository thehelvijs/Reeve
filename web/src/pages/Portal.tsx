import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { api, endpointString, type Collection, type CollectionRef, type Host, type Tool } from '../api';
import { useAuth } from '../auth';
import { Button, Card, Pill } from '../components/ui';
import StatusPill from '../components/StatusPill';
import SearchBar from '../components/SearchBar';
import EntityIcon from '../components/EntityIcon';
import LeafletMap from '../components/LeafletMap';
import PortalGraph from '../components/PortalGraph';
import CollectionInfoModal from '../components/CollectionInfoModal';
import HostInfoModal from '../components/HostInfoModal';
import ToolInfoModal from '../components/ToolInfoModal';
import VisibilityToggle from '../components/VisibilityToggle';
import ThemeToggle from '../components/ThemeToggle';
import { groupToolsByCollection, groupToolsByHost } from '../lib/group';
import { SOURCE_URL, UI_VERSION } from '../version';

type View = 'list' | 'map' | 'graph';

type GroupBy = 'collection' | 'host';

type Section =
  | { kind: 'host'; key: string; host: Host | null; tools: Tool[] }
  | { kind: 'collection'; key: string; collection: CollectionRef | null; tools: Tool[] };

type Selection =
  | { kind: 'host'; host: Host }
  | { kind: 'tool'; tool: Tool }
  | { kind: 'collection'; collection: CollectionRef };

export default function Portal() {
  const { user, loading } = useAuth();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const [tools, setTools] = useState<Tool[]>([]);
  const [hosts, setHosts] = useState<Host[]>([]);
  const [collections, setCollections] = useState<Collection[]>([]);
  const [search, setSearch] = useState('');
  const [view, setView] = useState<View>('list');
  const [groupBy, setGroupBy] = useState<GroupBy>(() => {
    if (params.get('by') === 'host') {
      return 'host';
    }
    return 'collection';
  });
  const [selected, setSelected] = useState<Selection | null>(null);

  useEffect(() => {
    if (loading) {
      return;
    }
    const toolsPath = user ? '/api/tools' : '/api/public/tools';
    const hostsPath = user ? '/api/hosts' : '/api/public/hosts';
    const collectionsPath = user ? '/api/collections' : '/api/public/collections';
    const load = () => {
      api.get<Tool[]>(toolsPath).then((t) => setTools(t ?? [])).catch(() => setTools([]));
      api.get<Host[]>(hostsPath).then((h) => setHosts(h ?? [])).catch(() => setHosts([]));
      api
        .get<Collection[]>(collectionsPath)
        .then((c) => setCollections(c ?? []))
        .catch(() => setCollections([]));
    };
    load();
    const poll = setInterval(load, 15000);
    return () => clearInterval(poll);
  }, [user, loading]);

  const changeGroupBy = (next: GroupBy) => {
    setGroupBy(next);
    const p = new URLSearchParams(params);
    if (next === 'host') {
      p.set('by', 'host');
    } else {
      p.delete('by');
    }
    setParams(p, { replace: true });
  };

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) {
      return tools;
    }
    return tools.filter((t) => t.name.toLowerCase().includes(q) || t.description.toLowerCase().includes(q));
  }, [tools, search]);

  const sections = useMemo<Section[]>(() => {
    if (groupBy === 'host') {
      return groupToolsByHost(filtered, hosts).map((g) => ({
        kind: 'host',
        key: g.host?.id ?? 'unassigned',
        host: g.host,
        tools: g.tools,
      }));
    }
    return groupToolsByCollection(filtered).map((g) => ({
      kind: 'collection',
      key: g.collection?.id ?? 'ungrouped',
      collection: g.collection,
      tools: g.tools,
    }));
  }, [groupBy, filtered, hosts]);

  const describe = (id: string) => collections.find((c) => c.id === id)?.description;

  const openTool = (id: string) => {
    const t = tools.find((x) => x.id === id);
    if (t) {
      setSelected({ kind: 'tool', tool: t });
    }
  };
  const replaceTool = (next: Tool) => setTools((all) => all.map((t) => (t.id === next.id ? next : t)));

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
          <span className="rounded-button bg-accent px-1.5 text-sm font-semibold text-accent-fg">Reeve</span>
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
        <div className="ml-auto flex shrink-0 items-center gap-2">
          <ThemeToggle />
          <Button onClick={() => navigate(user ? '/dashboard' : '/login')}>Dashboard</Button>
        </div>
      </header>

      <main className="flex min-h-0 flex-1 flex-col px-8 py-6">
        <div className="flex min-h-0 w-full flex-1 flex-col">
        <div className="mb-4 flex shrink-0 justify-center gap-2">
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
          {view === 'list' && (
            <div className="inline-flex rounded-button border border-hairline p-0.5">
              {(['collection', 'host'] as GroupBy[]).map((b) => (
                <button
                  key={b}
                  type="button"
                  onClick={() => changeGroupBy(b)}
                  className={`rounded-[5px] px-3 py-1 text-sm capitalize transition-colors ${
                    groupBy === b ? 'bg-surface-2 text-content' : 'text-muted hover:text-content'
                  }`}
                >
                  {b}
                </button>
              ))}
            </div>
          )}
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
            {sections.length === 0 && <p className="text-sm text-muted">No tools to show.</p>}
            {sections.map((g) => (
          <section key={g.key} className="mb-8">
            {g.kind === 'host' && g.host && (
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
            )}
            {g.kind === 'host' && !g.host && (
              <div className="flex items-center gap-2 px-1">
                <HostDot status={undefined} />
                <h2 className="text-sm font-medium text-content">Unassigned</h2>
                <span className="text-xs text-muted">{g.tools.length}</span>
              </div>
            )}
            {g.kind === 'collection' && g.collection && (
              <button
                type="button"
                onClick={() => setSelected({ kind: 'collection', collection: g.collection! })}
                className="flex items-center gap-2 rounded-button px-1 py-0.5 transition-colors hover:bg-surface-2"
              >
                <EntityIcon url={g.collection.icon_url} name={g.collection.name} size={22} />
                <h2 className="text-sm font-medium text-content">{g.collection.name}</h2>
                <span className="text-xs text-muted">{g.tools.length}</span>
              </button>
            )}
            {g.kind === 'collection' && !g.collection && (
              <div className="flex items-center gap-2 px-1">
                <h2 className="text-sm font-medium text-content">Ungrouped</h2>
                <span className="text-xs text-muted">{g.tools.length}</span>
              </div>
            )}
            <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {/* The toggle is a button, so it sits beside the card's button
                  rather than inside it: a button within a button is invalid
                  HTML that the parser silently splits, and the card then stops
                  responding at all. */}
              {g.tools.map((t) => (
                <Card key={t.id} className="relative flex h-full flex-col px-4 py-3 transition-colors hover:bg-surface-2">
                  <div className="absolute right-4 top-3 flex items-center gap-1.5">
                    <VisibilityToggle tool={t} onChanged={replaceTool} />
                    <StatusPill status={t.status} />
                  </div>
                  <button type="button" onClick={() => openTool(t.id)} className="flex h-full flex-col text-left">
                    <EntityIcon url={t.icon_url} name={t.name} size={36} />
                    <p className="mt-2 w-full truncate text-sm font-medium text-content">{t.name}</p>
                    <p className="w-full truncate font-mono text-xs text-muted">{endpointString(t) || '—'}</p>
                    {t.collections.length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1.5">
                        {t.collections.map((c) => (
                          <Pill key={c.id}>{c.name}</Pill>
                        ))}
                      </div>
                    )}
                  </button>
                </Card>
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
      {selected?.kind === 'collection' && (
        <CollectionInfoModal
          collection={{ ...selected.collection, description: describe(selected.collection.id) }}
          tools={tools.filter((t) => t.collections.some((c) => c.id === selected.collection.id))}
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
            navigate(`/services/${selected.tool.id}`);
          }}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  );
}

function HostDot({ status }: { status?: Host['status'] }) {
  let color = 'bg-idle-solid';
  if (status === 'online') {
    color = 'bg-up-solid';
  } else if (status === 'offline') {
    color = 'bg-down-solid';
  }
  return <span className={`inline-block h-2 w-2 rounded-full ${color}`} />;
}
