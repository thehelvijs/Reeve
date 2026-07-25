import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  api,
  type Host,
  type HostInventory as Inv,
  type InventoryItem,
  type ThresholdsPayload,
} from '../api';
import { useAuth } from '../auth';
import { Button, Card, ErrorText, Pill } from '../components/ui';
import BackLink from '../components/BackLink';
import EntityIcon from '../components/EntityIcon';
import IconUploader from '../components/IconUploader';
import ThumbnailUploader from '../components/ThumbnailUploader';
import MapPicker from '../components/MapPicker';
import HostMetrics from '../components/HostMetrics';
import EventHistory from '../components/EventHistory';
import UptimeSummary from '../components/UptimeSummary';
import ThresholdFields, {
  EMPTY_ROW,
  METRIC_KEYS,
  thresholdToDisplay,
  thresholdToWire,
  type ThresholdRows,
} from '../components/ThresholdFields';

const REFRESH_MS = 15000;

export default function HostInventory() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const [inv, setInv] = useState<Inv | null>(null);
  const [host, setHost] = useState<Host | null>(null);
  const [metricsKey, setMetricsKey] = useState(0);

  const load = useCallback(() => {
    api.get<Inv>(`/api/v1/hosts/${id}/inventory`).then(setInv);
    api.get<Host[]>('/api/v1/hosts').then((hs) => setHost((hs ?? []).find((h) => h.id === id) ?? null));
  }, [id]);

  useEffect(() => {
    load();
    const t = setInterval(load, REFRESH_MS);
    return () => clearInterval(t);
  }, [load]);

  const createFrom = (item: InventoryItem) => {
    const q = new URLSearchParams({
      name: item.name,
      host_id: id ?? '',
      source_type: item.source_type,
      source_ref: item.source_ref,
    });
    navigate(`/catalog/new?${q}`);
  };

  const clearMetrics = async () => {
    if (!id || !window.confirm('Clear all stored metric history for this host? This cannot be undone.')) {
      return;
    }
    await api.del(`/api/v1/admin/hosts/${id}/metrics`);
    setMetricsKey((k) => k + 1);
  };

  if (!inv) {
    return <p className="text-sm text-muted">Loading…</p>;
  }

  return (
    <div>
      <BackLink to="/hosts">Hosts</BackLink>
      <div className="mt-3 flex items-center gap-3">
        <EntityIcon url={host?.icon_url} name={host?.name ?? 'Host'} size={36} />
        <h1 className="text-2xl font-semibold tracking-tight text-content">{host?.name ?? 'Host'}</h1>
      </div>
      {id && <UptimeSummary path={`/api/v1/hosts/${id}/uptime`} />}
      {host?.thumbnail_url && (
        <img
          src={host.thumbnail_url}
          alt=""
          className="mt-4 w-full max-w-lg rounded-card border border-hairline object-cover"
        />
      )}

      {id && user?.role === 'admin' && (
        <Card className="mt-6 p-5">
          <p className="text-sm font-medium text-content">Icon</p>
          <div className="mt-4">
            <IconUploader
              url={host?.icon_url}
              name={host?.name ?? 'Host'}
              path={`/api/v1/admin/hosts/${id}/icon`}
              onChange={load}
            />
          </div>
          <p className="mt-6 text-sm font-medium text-content">Thumbnail</p>
          <div className="mt-4">
            <ThumbnailUploader
              url={host?.thumbnail_url}
              path={`/api/v1/admin/hosts/${id}/thumbnail`}
              onChange={load}
            />
          </div>
        </Card>
      )}

      {id && host && user?.role === 'admin' && (
        <Card className="mt-4 p-5">
          <p className="text-sm font-medium text-content">Location</p>
          <p className="mt-1 text-xs text-muted">Where this host physically lives. Shown on the map view.</p>
          <div className="mt-4">
            <MapPicker
              hostId={id}
              location={host.physical_location ?? ''}
              latitude={host.latitude}
              longitude={host.longitude}
              onSaved={load}
            />
          </div>
        </Card>
      )}

      {id && (
        <div className="mt-6">
          {user?.role === 'admin' && (
            <div className="mb-2 flex justify-end">
              <Button variant="secondary" onClick={clearMetrics}>
                Clear metrics
              </Button>
            </div>
          )}
          <HostMetrics key={metricsKey} path={`/api/v1/hosts/${id}/metrics`} />
        </div>
      )}

      {id && <EventHistory path={`/api/v1/hosts/${id}/events`} />}

      {id && user?.role === 'admin' && <HostThresholds hostId={id} />}

      <Section title="Systemd units" items={inv.services} onCreate={createFrom} />
      <Section title="Containers" items={inv.containers} onCreate={createFrom} />
      <Section title="Cron jobs" items={inv.cron_jobs} onCreate={createFrom} />
    </div>
  );
}

function HostThresholds({ hostId }: { hostId: string }) {
  const [rows, setRows] = useState<ThresholdRows>({});
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);

  const load = useCallback(() => {
    api.get<ThresholdsPayload>(`/api/v1/admin/hosts/${hostId}/thresholds`).then((t) => {
      const next: ThresholdRows = {};
      for (const key of METRIC_KEYS) {
        const row = t.thresholds?.[key];
        if (row) {
          next[key] = { enabled: row.enabled, threshold: thresholdToDisplay(key, row.threshold) };
        } else {
          next[key] = { ...EMPTY_ROW };
        }
      }
      setRows(next);
    });
  }, [hostId]);

  useEffect(() => {
    load();
  }, [load]);

  const save = async () => {
    setError('');
    setSaved(false);
    const thresholds: ThresholdsPayload['thresholds'] = {};
    for (const key of METRIC_KEYS) {
      const row = rows[key];
      if (row && row.threshold !== '') {
        thresholds[key] = { enabled: row.enabled, threshold: thresholdToWire(key, row.threshold) };
      }
    }
    try {
      await api.put(`/api/v1/admin/hosts/${hostId}/thresholds`, { thresholds });
      setSaved(true);
      setTimeout(() => setSaved(false), 1500);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to save thresholds');
    }
  };

  const reset = async () => {
    setError('');
    setSaved(false);
    try {
      await api.del(`/api/v1/admin/hosts/${hostId}/thresholds`);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to reset thresholds');
    }
  };

  return (
    <div className="mt-6">
      <h2 className="text-sm font-medium text-content">Alert thresholds (this host)</h2>
      <Card className="mt-2 p-5">
        <p className="text-sm text-muted">
          Overrides the global default for all four metrics on this host.
        </p>
        <div className="mt-4">
          <ThresholdFields
            value={rows}
            onChange={(metric, row) => setRows((r) => ({ ...r, [metric]: row }))}
          />
        </div>
        <div className="mt-4 flex items-center gap-3">
          <Button onClick={save}>Save</Button>
          <Button variant="secondary" onClick={reset}>
            Reset to global
          </Button>
          {saved && <span className="text-sm text-muted">Saved.</span>}
        </div>
        <ErrorText>{error}</ErrorText>
      </Card>
    </div>
  );
}

function Section({
  title,
  items,
  onCreate,
}: {
  title: string;
  items: InventoryItem[];
  onCreate: (i: InventoryItem) => void;
}) {
  return (
    <div className="mt-6">
      <h2 className="text-sm font-medium text-content">
        {title} <span className="text-muted">({items.length})</span>
      </h2>
      <Card className="mt-2 max-h-80 divide-y divide-hairline overflow-y-auto">
        {items.length === 0 && <p className="px-4 py-3 text-sm text-muted">None reported.</p>}
        {items.map((it) => (
          <div key={it.source_ref} className="flex items-center justify-between px-4 py-2.5">
            <div className="min-w-0">
              <p className="truncate text-sm text-content">{it.name}</p>
              <p className="truncate font-mono text-xs text-muted">{it.detail}</p>
            </div>
            {it.linked ? (
              <Pill tone="up">linked</Pill>
            ) : (
              <Button variant="secondary" onClick={() => onCreate(it)}>
                Add for monitoring
              </Button>
            )}
          </div>
        ))}
      </Card>
    </div>
  );
}
