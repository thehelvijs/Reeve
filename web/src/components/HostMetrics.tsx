import { useEffect, useState, type ReactNode } from 'react';
import { api } from '../api';
import { useAuth } from '../auth';
import { cssVar, useTheme } from '../lib/theme';
import { Card, Section, Table, Td, Th, Tr } from './ui';
import Chart, { type Series } from './Chart';
import { fmtBytes, fmtRate } from '../lib/format';
import { perSecond } from '../lib/counters';
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

// segmented is the shared look of a small set of exclusive choices — a range, a
// window. Every one of them was its own class string, and two of them had no
// focus ring at all, so the set could not be reached from the keyboard.
function segmented(selected: boolean): string {
  const base =
    'rounded-button px-2 py-1 text-xs font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-link';
  if (selected) {
    return `${base} bg-surface-2 text-content`;
  }
  return `${base} text-muted hover:text-content`;
}

export default function HostMetrics({ path, action }: { path: string; action?: ReactNode }) {
  const [range, setRange] = useState('24h');
  const [points, setPoints] = useState<HostPoint[]>([]);
  const [containers, setContainers] = useState<ContainerPoint[]>([]);
  const [procs, setProcs] = useState<ProcessSample[]>([]);
  // The server omits processes and container series for a basic account, so the
  // card would render as an empty table rather than simply not being there.
  const admin = useAuth().user?.role === 'admin';
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
    { label: 'RX', color: PRIMARY, data: perSecond(points.map((p) => p.net_rx), xs) },
    { label: 'TX', color: SECONDARY, data: perSecond(points.map((p) => p.net_tx), xs) },
  ];

  const sensors = Array.from(new Set(points.flatMap((p) => Object.keys(p.temps ?? {}))));
  const temp: Series[] = sensors.map((sensor, i) => ({
    label: sensor,
    color: PALETTE[i % PALETTE.length],
    data: points.map((p) => p.temps?.[sensor] ?? null),
  }));

  const diskio: Series[] = [
    { label: 'read', color: PRIMARY, data: perSecond(points.map((p) => p.disk_read), xs) },
    { label: 'write', color: SECONDARY, data: perSecond(points.map((p) => p.disk_write), xs) },
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

  const ranges = (
    <div className="flex gap-1">
      {RANGES.map((r) => (
        <button
          key={r}
          type="button"
          aria-pressed={range === r}
          onClick={() => setRange(r)}
          className={segmented(range === r)}
        >
          {r}
        </button>
      ))}
    </div>
  );

  // One sample cannot be a line. uPlot draws nothing and pads the time axis out
  // to a span of years, which reads as a broken chart rather than a new host.
  const chartable = points.length >= 2;

  // The grid is items-start: a card whose lone series carries no legend would
  // otherwise stretch to the height of the multi-series card beside it and end
  // in a band of empty canvas.

  return (
    <div className="space-y-6">
      <Section
        title="Charts"
        action={
          <>
            {ranges}
            {action}
          </>
        }
      >
        {!chartable ? (
          <Card className="p-4">
            <p className="text-sm text-muted">
              {points.length === 0
                ? 'No sample in this range yet. The agent pushes one every ~15s.'
                : 'Only one sample in this range so far. A line needs two — try a longer range.'}
            </p>
          </Card>
        ) : (
          <div className="grid grid-cols-1 items-start gap-3 lg:grid-cols-2">
            <MetricCard title="CPU" value={fmtPct(latest(cpu))}>
              <Chart xs={xs} series={cpu} fmt={fmtPct} />
            </MetricCard>
            <MetricCard title="Memory" value={fmtBytes(latest(mem))}>
              <Chart xs={xs} series={mem} fmt={fmtBytes} />
            </MetricCard>
            <MetricCard title="Disk" value={fmtBytes(latest(disk))}>
              <Chart xs={xs} series={disk} fmt={fmtBytes} />
            </MetricCard>
            <MetricCard title="Network">
              <Chart xs={xs} series={net} fmt={fmtRate} />
            </MetricCard>
            {sensors.length > 0 && (
              <MetricCard title="Temperature">
                <Chart xs={xs} series={temp} fmt={(v) => `${v.toFixed(0)}°C`} />
              </MetricCard>
            )}
            <MetricCard title="Disk I/O">
              <Chart xs={xs} series={diskio} fmt={fmtRate} />
            </MetricCard>
            <MetricCard title="Load average">
              <Chart xs={xs} series={load} fmt={(v) => v.toFixed(2)} />
            </MetricCard>
            {hasGPU && (
              <MetricCard title="GPU" value={fmtPct(latest(gpuUtil))}>
                <Chart xs={xs} series={gpuUtil} fmt={fmtPct} />
              </MetricCard>
            )}
            {hasGPU && (
              <MetricCard title="GPU memory" value={fmtBytes(latest(gpuMem))}>
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
      </Section>

      <DiskList disks={disks} />

      {admin && <ProcessTable procs={procs} path={path} />}
    </div>
  );
}

// latest is the newest non-null reading of a single-series chart, which is the
// number an operator actually came for; the plot is the context around it.
function latest(series: Series[]): number {
  const data = series[0]?.data ?? [];
  for (let i = data.length - 1; i >= 0; i -= 1) {
    const v = data[i];
    if (v != null) {
      return v;
    }
  }
  return 0;
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

  // A sortable column says so: the header is the control, and the one in force
  // carries the arrow rather than leaving the reader to infer the order.
  const sortable = (label: string, key: ProcessSort) => {
    const active = by === key;
    let tone = 'text-muted hover:text-content';
    if (active) {
      tone = 'text-content';
    }
    return (
      <Th className="text-right">
        <button
          type="button"
          onClick={() => setBy(key)}
          aria-label={`Sort by ${label}`}
          className={`inline-flex items-center gap-1 text-eyebrow font-semibold uppercase transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-link ${tone}`}
        >
          {label}
          {active && <span aria-hidden>↓</span>}
        </button>
      </Th>
    );
  };

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

  let description = 'What was running at the agent’s last push.';
  if (window !== 'last') {
    description = `Averaged over the last ${window}, heaviest first.`;
  }

  return (
    <Section
      title="Top processes"
      description={description}
      action={
        <>
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
                aria-pressed={window === w}
                onClick={() => setWindow(w)}
                className={segmented(window === w)}
              >
                {w === 'last' ? 'last push' : w}
              </button>
            ))}
          </div>
        </>
      }
    >
      {empty ? (
        <Card className="p-4">
          <p className="text-sm text-muted">{emptyText}</p>
        </Card>
      ) : (
        <Table
          head={
            <>
              {window === 'last' && <Th>PID</Th>}
              {window === 'last' && <Th>User</Th>}
              <Th>Command</Th>
              {sortable(window === 'last' ? 'CPU' : 'CPU avg', 'cpu')}
              {window !== 'last' && <Th className="text-right">CPU peak</Th>}
              {sortable(window === 'last' ? 'Memory' : 'Memory avg', 'mem')}
              {window !== 'last' && <Th className="text-right">Memory peak</Th>}
            </>
          }
        >
          {window === 'last' &&
            rows.map((p) => (
              <Tr key={p.pid}>
                <Td className="font-mono text-xs tabular-nums text-muted">{p.pid}</Td>
                <Td className="text-xs text-muted">{p.user}</Td>
                <Td className="max-w-md truncate font-mono text-xs text-content" title={p.command}>
                  {p.command}
                </Td>
                <Td className="text-right tabular-nums text-content">{p.cpu_pct.toFixed(1)}%</Td>
                <Td className="text-right tabular-nums text-content">{fmtBytes(p.mem_rss)}</Td>
              </Tr>
            ))}
          {window !== 'last' &&
            usageRows.map((u) => (
              <Tr key={u.command}>
                <Td className="max-w-md truncate font-mono text-xs text-content" title={u.command}>
                  {u.command}
                </Td>
                <Td className="text-right tabular-nums text-content">{u.cpu_avg.toFixed(1)}%</Td>
                <Td className="text-right tabular-nums text-muted">{u.cpu_max.toFixed(1)}%</Td>
                <Td className="text-right tabular-nums text-content">{fmtBytes(u.mem_avg)}</Td>
                <Td className="text-right tabular-nums text-muted">{fmtBytes(u.mem_max)}</Td>
              </Tr>
            ))}
        </Table>
      )}
    </Section>
  );
}

// A single-series card carries its current reading in the header, so the number
// is legible without hovering the plot for uPlot's legend.
function MetricCard({ title, value, children }: { title: string; value?: string; children: ReactNode }) {
  return (
    <Card className="p-4">
      <div className="mb-2 flex items-baseline justify-between gap-2">
        <p className="text-xs font-medium text-muted">{title}</p>
        {value && <p className="text-sm font-medium tabular-nums text-content">{value}</p>}
      </div>
      {children}
    </Card>
  );
}
