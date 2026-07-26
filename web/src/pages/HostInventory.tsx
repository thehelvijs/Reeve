import { useCallback, useEffect, useState } from 'react';
import ConfirmModal from '../components/ConfirmModal';
import { needsConfirm, rowConfirmation } from '../lib/confirmText';
import { useNavigate, useParams } from 'react-router-dom';
import {
  api,
  type AutoUpdatePolicy,
  type Host,
  type HostInventory as Inv,
  type InventoryItem,
  type ThresholdsPayload,
} from '../api';
import { useAuth } from '../auth';
import { Button, Card, ErrorText, Field, Form, Input, Pill } from '../components/ui';
import Modal from '../components/Modal';
import { POLICY_LABEL, UPDATE_LABEL, UPDATE_TONE } from '../lib/agentUpdate';
import BackLink from '../components/BackLink';
import EntityIcon from '../components/EntityIcon';
import IconUploader from '../components/IconUploader';
import ThumbnailUploader from '../components/ThumbnailUploader';
import MapPicker from '../components/MapPicker';
import HostMetrics from '../components/HostMetrics';
import CredentialsSection from '../components/CredentialsSection';
import HostControls, { unavailableReason } from '../components/HostControls';
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
  const [clearing, setClearing] = useState(false);

  const load = useCallback(() => {
    api.get<Inv>(`/api/hosts/${id}/inventory`).then(setInv);
    api.get<Host[]>('/api/hosts').then((hs) => setHost((hs ?? []).find((h) => h.id === id) ?? null));
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

  // A row's buttons are live only for an admin, on a host whose agent said it
  // will run commands. controlFor returns null otherwise, and the rows render
  // exactly as they did before this feature existed.
  const controlFor = (kind: 'service' | 'container') => {
    if (!id || !host || user?.role !== 'admin' || unavailableReason(host) !== '') {
      return null;
    }
    return async (verb: string, target: string) => {
      await api.post(`/api/admin/hosts/${id}/commands`, { action: `${kind}_${verb}`, target });
    };
  };

  const clearMetrics = async () => {
    await api.del(`/api/admin/hosts/${id}/metrics`);
    setMetricsKey((k) => k + 1);
  };

  const hostLabel = host?.name ?? 'this host';

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
      {id && <UptimeSummary path={`/api/hosts/${id}/uptime`} />}
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
              path={`/api/admin/hosts/${id}/icon`}
              onChange={load}
            />
          </div>
          <p className="mt-6 text-sm font-medium text-content">Thumbnail</p>
          <div className="mt-4">
            <ThumbnailUploader
              url={host?.thumbnail_url}
              path={`/api/admin/hosts/${id}/thumbnail`}
              onChange={load}
            />
          </div>
        </Card>
      )}

      {id && <CredentialsSection hostId={id} canManage={user?.role === 'admin'} />}

      {id && host && user?.role === 'admin' && <HostControls hostId={id} host={host} />}

      {host && user?.role === 'admin' && <AgentCard host={host} onChanged={load} />}

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
              <Button variant="secondary" onClick={() => setClearing(true)}>
                Clear metrics
              </Button>
            </div>
          )}
          <HostMetrics key={metricsKey} path={`/api/hosts/${id}/metrics`} />
        </div>
      )}

      {clearing && (
        <ConfirmModal
          title="Clear stored metrics?"
          body={`Every metric sample, container stat and process average kept for ${hostLabel} is deleted. Current status and inventory stay. This cannot be undone.`}
          confirmLabel="Clear metrics"
          onConfirm={clearMetrics}
          onClose={() => setClearing(false)}
        />
      )}

      {id && <EventHistory path={`/api/hosts/${id}/events`} />}

      {id && user?.role === 'admin' && <HostThresholds hostId={id} />}

      <Section
        title="Systemd units"
        items={inv.services}
        onCreate={createFrom}
        control={controlFor('service')}
        hostName={hostLabel}
      />
      <Section
        title="Containers"
        items={inv.containers}
        onCreate={createFrom}
        control={controlFor('container')}
        hostName={hostLabel}
      />
      <Section title="Cron jobs" items={inv.cron_jobs} onCreate={createFrom} hostName={hostLabel} />

      {id && host && user?.role === 'admin' && <DeleteHost hostId={id} host={host} />}
    </div>
  );
}

function AgentCard({ host, onChanged }: { host: Host; onChanged: () => void }) {
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const setPolicy = async (policy: AutoUpdatePolicy) => {
    setError('');
    setBusy(true);
    try {
      await api.put(`/api/admin/hosts/${host.id}/auto-update`, { policy });
      onChanged();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not save the policy');
    } finally {
      setBusy(false);
    }
  };

  const updateNow = async () => {
    setError('');
    setBusy(true);
    try {
      await api.post(`/api/admin/hosts/${host.id}/update-now`, {});
      onChanged();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not start the update');
    } finally {
      setBusy(false);
    }
  };

  let versionText = 'never reported';
  if (host.agent_version) {
    versionText = host.agent_version;
  }

  // A non-off policy still landing on `disabled` means the host's own REEVE_AUTO_UPDATE=false vetoed it.
  let stateNote = '';
  if (host.update_state === 'disabled') {
    if (host.auto_update === 'off') {
      stateNote = 'This host will not self-update because its policy is set to never update. Change it below.';
    } else {
      stateNote =
        'This host will not self-update: the machine itself runs the agent with REEVE_AUTO_UPDATE=false, which the server cannot override.';
    }
  }
  if (host.update_state === 'unknown') {
    stateNote =
      'This host has not reported which agent binary it runs, so the server cannot tell whether it is current. Update now installs the published build anyway; the agent checks its own binary against it.';
  }

  return (
    <Card className="mt-4 p-5">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-content">Agent</p>
          <p className="mt-1 text-xs text-muted">
            Version {versionText}
            {host.ip_address && <> · reporting from <span className="font-mono">{host.ip_address}</span></>}
          </p>
        </div>
        <Pill tone={UPDATE_TONE[host.update_state]}>{UPDATE_LABEL[host.update_state]}</Pill>
      </div>

      {stateNote && <p className="mt-3 text-xs text-muted">{stateNote}</p>}

      <div className="mt-4 flex flex-wrap items-end gap-3">
        <Field label="Auto-update">
          <select
            value={host.auto_update}
            onChange={(e) => setPolicy(e.target.value as AutoUpdatePolicy)}
            disabled={busy}
            className="rounded-button border border-hairline bg-surface-2 px-3 py-2 text-sm text-content focus:outline-none focus:ring-2 focus:ring-accent"
          >
            {(['default', 'on', 'off'] as const).map((p) => (
              <option key={p} value={p}>
                {POLICY_LABEL[p]}
              </option>
            ))}
          </select>
        </Field>
        <Button variant="secondary" onClick={updateNow} disabled={busy}>
          Update now
        </Button>
      </div>
      <ErrorText>{error}</ErrorText>
    </Card>
  );
}

function HostThresholds({ hostId }: { hostId: string }) {
  const [rows, setRows] = useState<ThresholdRows>({});
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);

  const load = useCallback(() => {
    api.get<ThresholdsPayload>(`/api/admin/hosts/${hostId}/thresholds`).then((t) => {
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
      await api.put(`/api/admin/hosts/${hostId}/thresholds`, { thresholds });
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
      await api.del(`/api/admin/hosts/${hostId}/thresholds`);
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
        <Form onSubmit={save}>
          <div className="mt-4">
            <ThresholdFields
              value={rows}
              onChange={(metric, row) => setRows((r) => ({ ...r, [metric]: row }))}
            />
          </div>
          <div className="mt-4 flex items-center gap-3">
            <Button type="submit">Save</Button>
            <Button variant="secondary" onClick={reset}>
              Reset to global
            </Button>
            {saved && <span className="text-sm text-muted">Saved.</span>}
          </div>
          <ErrorText>{error}</ErrorText>
        </Form>
      </Card>
    </div>
  );
}

// RowControls queues one action for a single unit or container. It reports its
// own failure inline: a button that silently does nothing is worse than none.
// Stop and restart ask first — they take whatever is using the unit down with
// them, and these rows sit close enough together to hit the wrong one.
function RowControls({
  target,
  hostName,
  run,
}: {
  target: string;
  hostName: string;
  run: (verb: string, target: string) => Promise<void>;
}) {
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState('');
  const [confirming, setConfirming] = useState('');

  const fire = async (verb: string) => {
    setBusy(true);
    setFailed('');
    try {
      await run(verb, target);
    } catch (e) {
      setFailed(e instanceof Error ? e.message : 'failed');
      throw e;
    } finally {
      setBusy(false);
    }
  };

  const click = (verb: string) => {
    if (needsConfirm(verb)) {
      setConfirming(verb);
      return;
    }
    fire(verb).catch(() => undefined);
  };

  return (
    <div className="flex items-center gap-1">
      {failed && <span className="mr-1 text-xs text-red-400">{failed}</span>}
      {['start', 'stop', 'restart'].map((verb) => (
        <button
          key={verb}
          type="button"
          disabled={busy}
          onClick={() => click(verb)}
          className="rounded-button px-2 py-1 text-xs text-muted transition-colors hover:bg-surface-2 hover:text-content disabled:opacity-50"
        >
          {verb}
        </button>
      ))}
      {confirming && (
        <ConfirmModal
          {...rowConfirmation(confirming, target, hostName)}
          onConfirm={() => fire(confirming)}
          onClose={() => setConfirming('')}
        />
      )}
    </div>
  );
}

// DeleteHost removes the host and everything the schema hangs off it. The
// confirm names what goes with it, because credentials and history cascade and
// a catalogued service does not: tools keep a host_id that no longer resolves.
function DeleteHost({ hostId, host }: { hostId: string; host: Host }) {
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState(false);
  const [typed, setTyped] = useState('');
  const [error, setError] = useState('');

  const remove = async () => {
    setError('');
    try {
      await api.del(`/api/admin/hosts/${hostId}`);
      navigate('/hosts');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not delete this host');
    }
  };

  return (
    <Card className="mt-6 border-red-900/50 p-5">
      <p className="text-sm font-medium text-content">Delete host</p>
      <p className="mt-1 text-xs text-muted">
        Removes {host.name}, its stored credentials, metrics, events and command history. The agent
        on the machine keeps running until you uninstall it, and any service pinned to this host
        keeps a reference that no longer resolves.
      </p>
      <div className="mt-4">
        <Button variant="danger" onClick={() => setConfirming(true)}>
          Delete this host
        </Button>
      </div>
      <ErrorText>{error}</ErrorText>

      {confirming && (
        <Modal title={`Delete ${host.name}?`} onClose={() => setConfirming(false)}>
          <Form onSubmit={remove}>
            <p className="text-sm text-muted">
              This cannot be undone. Its credentials are destroyed with it. Type the host name to
              confirm.
            </p>
            <div className="mt-4">
              <Input
                value={typed}
                autoFocus
                placeholder={host.name}
                onChange={(e) => setTyped(e.target.value)}
              />
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <Button type="button" variant="secondary" onClick={() => setConfirming(false)}>
                Cancel
              </Button>
              <Button type="submit" variant="danger" disabled={typed !== host.name}>
                Delete
              </Button>
            </div>
          </Form>
        </Modal>
      )}
    </Card>
  );
}

function Section({
  title,
  items,
  onCreate,
  control,
  hostName,
}: {
  title: string;
  items: InventoryItem[];
  onCreate: (i: InventoryItem) => void;
  control?: ((verb: string, target: string) => Promise<void>) | null;
  hostName: string;
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
            <div className="flex shrink-0 items-center gap-2">
              {control && <RowControls target={it.source_ref} hostName={hostName} run={control} />}
              {it.linked ? (
                <Pill tone="up">linked</Pill>
              ) : (
                <Button variant="secondary" onClick={() => onCreate(it)}>
                  Add for monitoring
                </Button>
              )}
            </div>
          </div>
        ))}
      </Card>
    </div>
  );
}
