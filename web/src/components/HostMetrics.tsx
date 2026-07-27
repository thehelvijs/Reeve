import { useEffect, useState } from 'react';
import { api } from '../api';
import { cssVar, useTheme } from '../lib/theme';
import { Card } from './ui';
import Chart, { type Series } from './Chart';
import { fmtBytes } from '../lib/format';
import DiskList from './DiskList';
import SearchBar from './SearchBar';
import { matchesQuery } from '../lib/search';
import type { DiskUsage } from '../lib/disks';
import {
  sortProcesses,
  sortUsage,
  USAGE_WINDOWS,
  type ProcessSample,
  type ProcessSort,
  type ProcessUsage,
  type UsageWindow,
} from '../lib/processes';

interface HostPoint {
  ts: string;
  cpu_pct: number;
  mem_used: number;
  mem_total: number;
  disk_used: number;
  disk_total: number;
  disk_read: number;
  disk_write: number;
  net_rx: number;
  net_tx: number;
  load1: number;
  load5: number;
  load15: number;
  temps?: Record<string, number>;
  gpu_util: number;
  gpu_mem_used: number;
  gpu_mem_total: number;
}

interface ContainerPoint {
  container_id: string;
  name: string;
  ts: string;
  cpu_pct: number;
  mem_used: number;
  mem_limit: number;
}

function labelFor(containers: ContainerPoint[], id: string): string {
  const sample = containers.find((c) => c.container_id === id);
  if (sample && sample.name !== '') {
    return sample.name;
  }
  return id.slice(0, 12);
}

const RANGES = ['1h', '12h', '24h', '7d', '30d'];

// Slots, not colors: --series-1..8 are set per theme in theme.css, each set
// validated against its own canvas. A slot is assigned by position and never by
// rank, so a series keeps its color when the drawn count changes.
//
// A chart with one line takes --data-primary instead, which is the brand lime on
// dark. Lime is too bright to sit in an even categorical ramp, but a lone line has
// no ramp to be even with.
const SERIES_SLOTS = 8;

function palette(): string[] {
  const out: string[] = [];
  for (let i = 1; i <= SERIES_SLOTS; i += 1) {
    out.push(cssVar(`--series-${i}`));
  }
  return out;
}

const fmtPct = (v: number) => `${v.toFixed(0)}%`;

export default function HostMetrics({ path }: { path: string }) {
  const [range, setRange] = useState('24h');
  const [points, setPoints] = useState<HostPoint[]>([]);
  const [containers, setContainers] = useState<ContainerPoint[]>([]);
  const [procs, setProcs] = useState<ProcessSample[]>([]);
  const [disks, setDisks] = useState<DiskUsage[]>([]);
  // Subscribed for the re-render, not for the value: a theme switch re-runs
  // palette(), which reads the slots the new theme set.
  useTheme();
  const PALETTE = palette();
  const SOLO = cssVar('--data-primary');
  const PRIMARY = PALETTE[0];
  const SECONDARY = PALETTE[1];

  useEffect(() => {
    const load = () => {
      api
        .get<{
          host: HostPoint[];
          containers: ContainerPoint[];
          processes?: { procs: ProcessSample[] };
          disks?: { disks: DiskUsage[] };
        }>(`${path}?range=${range}`)
        .then((d) => {
          setPoints(d.host ?? []);
          setContainers(d.containers ?? []);
          setProcs(d.processes?.procs ?? []);
          setDisks(d.disks?.disks ?? []);
        });
    };
    load();
    const poll = setInterval(load, 15000);
    return () => clearInterval(poll);
  }, [path, range]);

  const xs = points.map((p) => Math.floor(new Date(p.ts).getTime() / 1000));
  const cpu: Series[] = [{ label: 'CPU %', color: SOLO, data: points.map((p) => p.cpu_pct) }];
  const mem: Series[] = [{ label: 'Memory used', color: SOLO, data: points.map((p) => p.mem_used) }];
  const disk: Series[] = [{ label: 'Disk used', color: SOLO, data: points.map((p) => p.disk_used) }];
  const net: Series[] = [
    { label: 'RX', color: PRIMARY, data: points.map((p) => p.net_rx) },
    { label: 'TX', color: SECONDARY, data: points.map((p) => p.net_tx) },
  ];

  const sensors = Array.from(new Set(points.flatMap((p) => Object.keys(p.temps ?? {}))));
  const temp: Series[] = sensors.map((sensor, i) => ({
    label: sensor,
    color: PALETTE[i % PALETTE.length],
    data: points.map((p) => p.temps?.[sensor] ?? null),
  }));

  const diskio: Series[] = [
    { label: 'read', color: PRIMARY, data: points.map((p) => p.disk_read) },
    { label: 'write', color: SECONDARY, data: points.map((p) => p.disk_write) },
  ];

  const load: Series[] = [
    { label: '1m', color: PRIMARY, data: points.map((p) => p.load1) },
    { label: '5m', color: PALETTE[1], data: points.map((p) => p.load5) },
    { label: '15m', color: PALETTE[2], data: points.map((p) => p.load15) },
  ];

  const hasGPU = points.some((p) => p.gpu_mem_total > 0);
  const gpuUtil: Series[] = [{ label: 'GPU %', color: SOLO, data: points.map((p) => p.gpu_util) }];
  const gpuMem: Series[] = [{ label: 'GPU memory', color: SOLO, data: points.map((p) => p.gpu_mem_used) }];

  const containerIds = Array.from(new Set(containers.map((c) => c.container_id)));
  const containerCpu: Series[] = containerIds.map((id, i) => {
    const byTs = new Map(
      containers
        .filter((c) => c.container_id === id)
        .map((c) => [Math.floor(new Date(c.ts).getTime() / 1000), c.cpu_pct]),
    );
    return {
      label: labelFor(containers, id),
      color: PALETTE[i % PALETTE.length],
      data: xs.map((x) => byTs.get(x) ?? null),
    };
  });
  const containerMem: Series[] = containerIds.map((id, i) => {
    const byTs = new Map(
      containers
        .filter((c) => c.container_id === id)
        .map((c) => [Math.floor(new Date(c.ts).getTime() / 1000), c.mem_used]),
    );
    return {
      label: labelFor(containers, id),
      color: PALETTE[i % PALETTE.length],
      data: xs.map((x) => byTs.get(x) ?? null),
    };
  });

  return (
    <div>
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-medium text-content">Metrics</h2>
        <div className="flex gap-1">
          {RANGES.map((r) => (
            <button
              key={r}
              onClick={() => setRange(r)}
              className={`rounded-button px-2 py-1 text-xs transition-colors ${
                range === r ? 'bg-surface-2 text-content' : 'text-muted hover:text-content'
              }`}
            >
              {r}
            </button>
          ))}
        </div>
      </div>

      {points.length === 0 ? (
        <Card className="mt-2 p-5">
          <p className="text-sm text-muted">No samples in this range yet.</p>
        </Card>
      ) : (
        <div className="mt-2 grid grid-cols-1 gap-3 lg:grid-cols-2">
          <MetricCard title="CPU"><Chart xs={xs} series={cpu} fmt={fmtPct} /></MetricCard>
          <MetricCard title="Memory"><Chart xs={xs} series={mem} fmt={fmtBytes} /></MetricCard>
          <MetricCard title="Disk"><Chart xs={xs} series={disk} fmt={fmtBytes} /></MetricCard>
          <MetricCard title="Network"><Chart xs={xs} series={net} fmt={fmtBytes} /></MetricCard>
          {sensors.length > 0 && (
            <MetricCard title="Temperature">
              <Chart xs={xs} series={temp} fmt={(v) => `${v.toFixed(0)}°C`} />
            </MetricCard>
          )}
          <MetricCard title="Disk I/O"><Chart xs={xs} series={diskio} fmt={fmtBytes} /></MetricCard>
          <MetricCard title="Load average">
            <Chart xs={xs} series={load} fmt={(v) => v.toFixed(2)} />
          </MetricCard>
          {hasGPU && (
            <MetricCard title="GPU">
              <Chart xs={xs} series={gpuUtil} fmt={fmtPct} />
            </MetricCard>
          )}
          {hasGPU && (
            <MetricCard title="GPU memory">
              <Chart xs={xs} series={gpuMem} fmt={fmtBytes} />
            </MetricCard>
          )}
          {containerCpu.length > 0 && (
            <MetricCard title="Container CPU">
              <Chart xs={xs} series={containerCpu} fmt={fmtPct} />
            </MetricCard>
          )}
          {containerMem.length > 0 && (
            <MetricCard title="Container memory">
              <Chart xs={xs} series={containerMem} fmt={fmtBytes} />
            </MetricCard>
          )}
        </div>
      )}

      <DiskList disks={disks} />

      <ProcessTable procs={procs} path={path} />
    </div>
  );
}

// ProcessTable answers two different questions from one card: what is running
// right now, and what a machine has actually been spending itself on over a
// window. The second is the one that finds waste, since a single push catches
// whatever happened to be busy in that instant.
function ProcessTable({ procs, path }: { procs: ProcessSample[]; path: string }) {
  const [by, setBy] = useState<ProcessSort>('cpu');
  const [window, setWindow] = useState<UsageWindow>('last');
  const [search, setSearch] = useState('');
  const [usage, setUsage] = useState<ProcessUsage[]>([]);
  const usagePath = path.replace(/\/metrics$/, '/process-usage');

  useEffect(() => {
    if (window === 'last') {
      return;
    }
    let live = true;
    const load = () => {
      api
        .get<{ processes: ProcessUsage[] }>(`${usagePath}?window=${window}`)
        .then((d) => {
          if (live) {
            setUsage(d.processes ?? []);
          }
        })
        .catch(() => undefined);
    };
    load();
    const poll = setInterval(load, 30000);
    return () => {
      live = false;
      clearInterval(poll);
    };
  }, [usagePath, window]);

  const heading = (label: string, key: ProcessSort) => {
    let className = 'text-right font-medium hover:text-content';
    if (by === key) {
      className = 'text-right font-medium text-content';
    }
    return (
      <th scope="col" className="sticky top-0 z-10 bg-surface-1 py-2 pl-3">
        <button type="button" onClick={() => setBy(key)} className={className}>
          {label}
        </button>
      </th>
    );
  };
  const plain = (label: string) => (
    <th scope="col" className="sticky top-0 z-10 bg-surface-1 py-2 pr-3 text-left font-medium">
      {label}
    </th>
  );

  const rows = sortProcesses(procs, by).filter((p) => matchesQuery(search, p.command, p.user, String(p.pid)));
  const usageRows = sortUsage(usage, by).filter((u) => matchesQuery(search, u.command));
  let empty = usageRows.length === 0;
  if (window === 'last') {
    empty = rows.length === 0;
  }
  let emptyText = 'Nothing reported for this window yet.';
  if (search.trim()) {
    emptyText = 'No process matches the search.';
  }

  return (
    <Card className="mt-4 p-4">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <p className="text-xs font-medium text-muted">
          Top processes{' '}
          <span className="text-muted">
            {window === 'last' ? '(at the last push)' : `(average over the last ${window})`}
          </span>
        </p>
        <div className="flex flex-wrap items-center gap-2">
          <SearchBar
            value={search}
            onChange={setSearch}
            placeholder="Search processes…"
            shortcut={false}
            className="w-56"
          />
          <div className="inline-flex rounded-button border border-hairline p-0.5">
            {USAGE_WINDOWS.map((w) => (
              <button
                key={w}
                type="button"
                onClick={() => setWindow(w)}
                className={`rounded-[5px] px-2 py-0.5 text-xs transition-colors ${
                  window === w ? 'bg-surface-2 text-content' : 'text-muted hover:text-content'
                }`}
              >
                {w === 'last' ? 'last push' : w}
              </button>
            ))}
          </div>
        </div>
      </div>
      {empty && <p className="py-2 text-sm text-muted">{emptyText}</p>}
      {!empty && (
        <div className="max-h-80 overflow-auto">
          <table className="w-full text-sm">
            <thead className="text-xs text-muted">
              <tr>
                {window === 'last' && plain('PID')}
                {window === 'last' && plain('User')}
                {plain('Command')}
                {heading(window === 'last' ? 'CPU' : 'CPU avg', 'cpu')}
                {window !== 'last' && plain('CPU peak')}
                {heading(window === 'last' ? 'Memory' : 'Memory avg', 'mem')}
                {window !== 'last' && plain('Memory peak')}
              </tr>
            </thead>
            <tbody>
              {window === 'last' &&
                rows.map((p) => (
                  <tr key={p.pid} className="border-t border-hairline">
                    <td className="py-1.5 pr-3 font-mono text-xs text-muted">{p.pid}</td>
                    <td className="py-1.5 pr-3 text-xs text-muted">{p.user}</td>
                    <td className="max-w-md truncate py-1.5 pr-3 font-mono text-xs text-content" title={p.command}>
                      {p.command}
                    </td>
                    <td className="py-1.5 pl-3 text-right tabular-nums text-content">{p.cpu_pct.toFixed(1)}%</td>
                    <td className="py-1.5 pl-3 text-right tabular-nums text-content">{fmtBytes(p.mem_rss)}</td>
                  </tr>
                ))}
              {window !== 'last' &&
                usageRows.map((u) => (
                  <tr key={u.command} className="border-t border-hairline">
                    <td className="max-w-md truncate py-1.5 pr-3 font-mono text-xs text-content" title={u.command}>
                      {u.command}
                    </td>
                    <td className="py-1.5 pl-3 text-right tabular-nums text-content">{u.cpu_avg.toFixed(1)}%</td>
                    <td className="py-1.5 pl-3 text-right tabular-nums text-muted">{u.cpu_max.toFixed(1)}%</td>
                    <td className="py-1.5 pl-3 text-right tabular-nums text-content">{fmtBytes(u.mem_avg)}</td>
                    <td className="py-1.5 pl-3 text-right tabular-nums text-muted">{fmtBytes(u.mem_max)}</td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

function MetricCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card className="p-4">
      <p className="mb-2 text-xs font-medium text-muted">{title}</p>
      {children}
    </Card>
  );
}
