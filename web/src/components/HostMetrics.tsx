import { useEffect, useState } from 'react';
import { api } from '../api';
import { Card } from './ui';
import Chart, { type Series } from './Chart';
import { fmtBytes } from '../lib/format';
import { sortProcesses, type ProcessSample, type ProcessSort } from '../lib/processes';

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
const ACCENT = '#e4f222';
const MUTED = '#8a8f98';
const PALETTE = ['#e4f222', '#7dd3fc', '#f0abfc', '#86efac', '#fca5a5', '#fdba74', '#c4b5fd', '#67e8f9'];

const fmtPct = (v: number) => `${v.toFixed(0)}%`;

export default function HostMetrics({ path }: { path: string }) {
  const [range, setRange] = useState('24h');
  const [points, setPoints] = useState<HostPoint[]>([]);
  const [containers, setContainers] = useState<ContainerPoint[]>([]);
  const [procs, setProcs] = useState<ProcessSample[]>([]);

  useEffect(() => {
    const load = () => {
      api
        .get<{
          host: HostPoint[];
          containers: ContainerPoint[];
          processes?: { procs: ProcessSample[] };
        }>(`${path}?range=${range}`)
        .then((d) => {
          setPoints(d.host ?? []);
          setContainers(d.containers ?? []);
          setProcs(d.processes?.procs ?? []);
        });
    };
    load();
    const poll = setInterval(load, 15000);
    return () => clearInterval(poll);
  }, [path, range]);

  const xs = points.map((p) => Math.floor(new Date(p.ts).getTime() / 1000));
  const cpu: Series[] = [{ label: 'CPU %', color: ACCENT, data: points.map((p) => p.cpu_pct) }];
  const mem: Series[] = [{ label: 'Memory used', color: ACCENT, data: points.map((p) => p.mem_used) }];
  const disk: Series[] = [{ label: 'Disk used', color: ACCENT, data: points.map((p) => p.disk_used) }];
  const net: Series[] = [
    { label: 'RX', color: ACCENT, data: points.map((p) => p.net_rx) },
    { label: 'TX', color: MUTED, data: points.map((p) => p.net_tx) },
  ];

  const sensors = Array.from(new Set(points.flatMap((p) => Object.keys(p.temps ?? {}))));
  const temp: Series[] = sensors.map((sensor, i) => ({
    label: sensor,
    color: PALETTE[i % PALETTE.length],
    data: points.map((p) => p.temps?.[sensor] ?? null),
  }));

  const diskio: Series[] = [
    { label: 'read', color: ACCENT, data: points.map((p) => p.disk_read) },
    { label: 'write', color: MUTED, data: points.map((p) => p.disk_write) },
  ];

  const load: Series[] = [
    { label: '1m', color: ACCENT, data: points.map((p) => p.load1) },
    { label: '5m', color: PALETTE[1], data: points.map((p) => p.load5) },
    { label: '15m', color: PALETTE[2], data: points.map((p) => p.load15) },
  ];

  const hasGPU = points.some((p) => p.gpu_mem_total > 0);
  const gpuUtil: Series[] = [{ label: 'GPU %', color: ACCENT, data: points.map((p) => p.gpu_util) }];
  const gpuMem: Series[] = [{ label: 'GPU memory', color: ACCENT, data: points.map((p) => p.gpu_mem_used) }];

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

      <ProcessTable procs={procs} />
    </div>
  );
}

// ProcessTable shows the latest snapshot, so it carries no range control: the
// numbers are as of the host's last push, whatever window the charts show.
function ProcessTable({ procs }: { procs: ProcessSample[] }) {
  const [by, setBy] = useState<ProcessSort>('cpu');
  if (procs.length === 0) {
    return null;
  }
  const rows = sortProcesses(procs, by);

  const heading = (label: string, key: ProcessSort) => {
    let className = 'text-right font-medium hover:text-content';
    if (by === key) {
      className = 'text-right font-medium text-content';
    }
    return (
      <th scope="col" className="py-2 pl-3">
        <button type="button" onClick={() => setBy(key)} className={className}>
          {label}
        </button>
      </th>
    );
  };

  return (
    <Card className="mt-4 p-4">
      <p className="mb-2 text-xs font-medium text-muted">
        Top processes <span className="text-muted">(at the last push)</span>
      </p>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="text-xs text-muted">
            <tr className="border-b border-hairline">
              <th scope="col" className="py-2 pr-3 text-left font-medium">PID</th>
              <th scope="col" className="py-2 pr-3 text-left font-medium">User</th>
              <th scope="col" className="py-2 pr-3 text-left font-medium">Command</th>
              {heading('CPU', 'cpu')}
              {heading('Memory', 'mem')}
            </tr>
          </thead>
          <tbody>
            {rows.map((p) => (
              <tr key={p.pid} className="border-b border-hairline last:border-0">
                <td className="py-1.5 pr-3 font-mono text-xs text-muted">{p.pid}</td>
                <td className="py-1.5 pr-3 text-xs text-muted">{p.user}</td>
                <td className="max-w-md truncate py-1.5 pr-3 font-mono text-xs text-content" title={p.command}>
                  {p.command}
                </td>
                <td className="py-1.5 pl-3 text-right tabular-nums text-content">{p.cpu_pct.toFixed(1)}%</td>
                <td className="py-1.5 pl-3 text-right tabular-nums text-content">{fmtBytes(p.mem_rss)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
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
