import { useEffect, useRef, useState } from 'react';
import { api } from '../api';
import { Button, Card, ErrorText } from '../components/ui';
import PageHeader from '../components/PageHeader';
import MetricBar from '../components/MetricBar';
import HostMetrics from '../components/HostMetrics';
import { fmtBytes, fmtUptime } from '../lib/format';

interface ServerInfo {
  version: string;
  started_at: string;
  uptime_secs: number;
  go: { version: string; os: string; arch: string; num_cpu: number; num_goroutine: number };
  db: { path: string; size_bytes: number; restore_staged: boolean };
  counts: { tools: number; hosts: number; users: number };
  host: { cpu_pct: number; mem_used: number; mem_total: number; disk_used: number; disk_total: number; uptime_secs: number };
}

export default function AdminServerInfo() {
  const [info, setInfo] = useState<ServerInfo | null>(null);

  const load = () => api.get<ServerInfo>('/api/admin/server-info').then(setInfo).catch(() => setInfo(null));
  useEffect(() => {
    load();
  }, []);

  if (!info) {
    return <p className="text-sm text-muted">Loading…</p>;
  }

  const pct = (used: number, total: number) => {
    if (total > 0) {
      return (used / total) * 100;
    }
    return 0;
  };

  return (
    <div>
      <PageHeader title="Server" subtitle="The machine and process running Reeve." />

      <Card className="mt-6 space-y-2 p-5">
        <MetricBar label="CPU" pct={info.host.cpu_pct} />
        <MetricBar
          label="Memory"
          pct={pct(info.host.mem_used, info.host.mem_total)}
          detail={`${fmtBytes(info.host.mem_used)} / ${fmtBytes(info.host.mem_total)}`}
        />
        <MetricBar
          label="Disk"
          pct={pct(info.host.disk_used, info.host.disk_total)}
          detail={`${fmtBytes(info.host.disk_used)} / ${fmtBytes(info.host.disk_total)}`}
        />
      </Card>

      <div className="mt-6">
        <HostMetrics path="/api/admin/server-metrics" />
      </div>

      <h2 className="mt-8 text-sm font-medium text-content">Details</h2>
      <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2">
        <Row label="Version" value={info.version} />
        <Row label="Uptime" value={fmtUptime(info.uptime_secs)} />
        <Row label="Go" value={`${info.go.version} · ${info.go.os}/${info.go.arch}`} />
        <Row label="Goroutines / CPUs" value={`${info.go.num_goroutine} / ${info.go.num_cpu}`} />
        <Row label="Database" value={`${fmtBytes(info.db.size_bytes)} · ${info.db.path}`} />
        <Row label="Tools / Hosts / Users" value={`${info.counts.tools} / ${info.counts.hosts} / ${info.counts.users}`} />
        <Row label="Host uptime" value={fmtUptime(info.host.uptime_secs)} />
      </div>

      <BackupSection staged={info.db.restore_staged} onChange={load} />
    </div>
  );
}

function BackupSection({ staged, onChange }: { staged: boolean; onChange: () => void }) {
  const fileInput = useRef<HTMLInputElement>(null);
  const [error, setError] = useState('');
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);

  const upload = async (file: File) => {
    setError('');
    setNote('');
    setBusy(true);
    const form = new FormData();
    form.append('backup', file);
    const res = await fetch('/api/admin/restore', { method: 'POST', credentials: 'include', body: form });
    const text = await res.text();
    const data = text ? JSON.parse(text) : null;
    setBusy(false);
    if (!res.ok) {
      setError(data?.message ?? 'restore failed');
      return;
    }
    setNote('Restore staged. Restart Reeve to swap this database in.');
    onChange();
  };

  const pick = () => {
    if (!window.confirm('Restoring replaces every account, service, and credential in this instance. Continue?')) {
      return;
    }
    fileInput.current?.click();
  };

  const cancel = async () => {
    setError('');
    setNote('');
    await api.del('/api/admin/restore');
    onChange();
  };

  return (
    <Card className="mt-8 p-5">
      <h2 className="text-sm font-medium text-content">Backup and restore</h2>
      <p className="mt-1 max-w-2xl text-xs text-muted">
        The backup is a consistent copy of the whole database. Credentials stay encrypted inside it — keep the master
        key somewhere else, or the backup cannot be read.
      </p>

      {staged && (
        <div className="mt-4 rounded-button border border-hairline bg-surface-2 px-3 py-2 text-xs text-content">
          A restore is staged and will be applied on the next start.
          <button className="ml-2 underline hover:text-accent" onClick={cancel}>
            discard it
          </button>
        </div>
      )}

      <div className="mt-4 flex flex-wrap items-center gap-2">
        <a
          href="/api/admin/backup"
          className="rounded-button border border-hairline px-3 py-1.5 text-sm text-content transition-colors hover:bg-surface-2"
        >
          Download backup
        </a>
        <Button variant="danger" disabled={busy} onClick={pick}>
          {busy ? 'Uploading…' : 'Restore from file'}
        </Button>
        <input
          ref={fileInput}
          type="file"
          accept=".db,application/vnd.sqlite3,application/octet-stream"
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0];
            e.target.value = '';
            if (f) {
              upload(f);
            }
          }}
        />
      </div>
      {note && <p className="mt-3 text-sm text-muted">{note}</p>}
      <div className="mt-3">
        <ErrorText>{error}</ErrorText>
      </div>
    </Card>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <Card className="p-4">
      <p className="text-xs text-muted">{label}</p>
      <p className="mt-1 truncate font-mono text-sm text-content">{value}</p>
    </Card>
  );
}
