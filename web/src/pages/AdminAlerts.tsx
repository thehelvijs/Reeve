import { useEffect, useState } from 'react';
import {
  api,
  type AlertEvent,
  type Delivery,
  type ThresholdsPayload,
} from '../api';
import { Button, Card, ErrorText, Field, Form, Input, Pill } from '../components/ui';
import PageHeader from '../components/PageHeader';
import ThresholdFields, {
  EMPTY_ROW,
  METRIC_KEYS,
  thresholdToDisplay,
  thresholdToWire,
  type ThresholdRows,
} from '../components/ThresholdFields';

function rowsFromPayload(thresholds: ThresholdsPayload['thresholds']): ThresholdRows {
  const rows: ThresholdRows = {};
  for (const key of METRIC_KEYS) {
    const t = thresholds[key];
    if (t) {
      rows[key] = { enabled: t.enabled, threshold: thresholdToDisplay(key, t.threshold) };
    } else {
      rows[key] = { ...EMPTY_ROW };
    }
  }
  return rows;
}

function deliveryTone(s: Delivery['status']): 'up' | 'muted' | 'down' {
  if (s === 'sent') {
    return 'up';
  }
  if (s === 'pending') {
    return 'muted';
  }
  return 'down';
}

function eventTone(resolved: boolean): 'up' | 'down' {
  if (resolved) {
    return 'up';
  }
  return 'down';
}

function eventPhase(resolved: boolean): string {
  if (resolved) {
    return 'resolved';
  }
  return 'firing';
}

function numOrEmpty(value: string): number | '' {
  if (value === '') {
    return '';
  }
  return Number(value);
}

export default function AdminAlerts() {
  const [events, setEvents] = useState<AlertEvent[]>([]);
  const [deliveries, setDeliveries] = useState<Delivery[]>([]);
  const [windowMin, setWindowMin] = useState<number | ''>(5);
  const [rows, setRows] = useState<ThresholdRows>({});
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    api.get<AlertEvent[]>('/api/v1/admin/alerts').then((e) => setEvents(e ?? []));
    api.get<Delivery[]>('/api/v1/admin/deliveries').then((d) => setDeliveries(d ?? []));
    api.get<ThresholdsPayload>('/api/v1/admin/thresholds').then((t) => {
      setWindowMin(Math.max(1, Math.round((t.window_secs ?? 300) / 60)));
      setRows(rowsFromPayload(t.thresholds ?? {}));
    });
  }, []);

  const saveThresholds = async () => {
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
      await api.put('/api/v1/admin/thresholds', {
        window_secs: windowMin === '' ? 300 : windowMin * 60,
        thresholds,
      });
      setSaved(true);
      setTimeout(() => setSaved(false), 1500);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to save thresholds');
    }
  };

  return (
    <div>
      <PageHeader title="Alerts" subtitle="Fired alerts and webhook delivery status." />

      <h2 className="mt-8 text-sm font-medium text-content">Thresholds</h2>
      <Card className="mt-2 p-5">
        <p className="text-sm text-muted">
          Sustained resource usage above these limits fires an alert.
        </p>
        <Form onSubmit={saveThresholds}>
          <div className="mt-4 max-w-xs">
            <Field label="Window (minutes)">
              <Input
                type="number"
                min={1}
                value={windowMin}
                onChange={(e) => setWindowMin(numOrEmpty(e.target.value))}
              />
            </Field>
          </div>
          <div className="mt-4">
            <ThresholdFields
              value={rows}
              onChange={(metric, row) => setRows((r) => ({ ...r, [metric]: row }))}
            />
          </div>
          <div className="mt-4 flex items-center gap-3">
            <Button type="submit">Save</Button>
            {saved && <span className="text-sm text-muted">Saved.</span>}
          </div>
          <ErrorText>{error}</ErrorText>
        </Form>
      </Card>

      <h2 className="mt-6 text-sm font-medium text-content">Recent alerts</h2>
      <Card className="mt-2 divide-y divide-hairline">
        {events.length === 0 && <p className="px-4 py-3 text-sm text-muted">No alerts fired.</p>}
        {events.map((e) => (
          <div key={e.id} className="flex items-center justify-between px-4 py-2.5 text-sm">
            <div className="flex items-center gap-2">
              <Pill tone={eventTone(Boolean(e.resolved_at))}>{e.type.replace('_', ' ')}</Pill>
              <span className="text-content">{e.message}</span>
            </div>
            <span className="text-xs text-muted">
              {eventPhase(Boolean(e.resolved_at))} · {new Date(e.fired_at).toLocaleString()}
            </span>
          </div>
        ))}
      </Card>

      <h2 className="mt-6 text-sm font-medium text-content">Webhook deliveries</h2>
      <Card className="mt-2 divide-y divide-hairline">
        {deliveries.length === 0 && <p className="px-4 py-3 text-sm text-muted">No deliveries yet.</p>}
        {deliveries.map((d) => (
          <div key={d.id} className="flex items-center justify-between px-4 py-2.5 text-sm">
            <div className="flex items-center gap-2">
              <Pill tone={deliveryTone(d.status)}>{d.status}</Pill>
              <span className="text-xs text-muted">attempts: {d.attempts}</span>
            </div>
            {d.last_error && <span className="truncate text-xs text-red-400">{d.last_error}</span>}
          </div>
        ))}
      </Card>
    </div>
  );
}
