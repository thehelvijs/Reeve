import { useCallback, useEffect, useState, type ReactNode } from 'react';
import ConfirmModal from '../components/ConfirmModal';
import SSHDeployModal from '../components/SSHDeployModal';
import AgentInstallModal from '../components/AgentInstallModal';
import { needsConfirm, rowConfirmation } from '../lib/confirmText';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import {
  api,
  type AutoUpdatePolicy,
  type Host,
  type HostInventory as Inv,
  type InventoryItem,
  type ThresholdsPayload,
} from '../api';
import { useAuth } from '../auth';
import {
  Button,
  Card,
  ErrorText,
  Facts,
  Field,
  Form,
  Input,
  Pill,
  Section,
  Select,
  Table,
  Tabs,
  Td,
  Th,
  Tr,
} from '../components/ui';
import Modal from '../components/Modal';
import EmptyState from '../components/EmptyState';
import { POLICY_LABEL, UPDATE_LABEL, UPDATE_TONE } from '../lib/agentUpdate';
import { hostRowActions } from '../lib/hostActions';
import { matchesQuery } from '../lib/search';
import { hostTone, unitStateTone } from '../lib/statusTone';
import { fmtBytes } from '../lib/format';
import SearchBar from '../components/SearchBar';
import EntityIcon from '../components/EntityIcon';
import IconUploader from '../components/IconUploader';
import ThumbnailUploader from '../components/ThumbnailUploader';
import MapPicker from '../components/MapPicker';
import MetricBar from '../components/MetricBar';
import HostMetrics from '../components/HostMetrics';
import CredentialsSection from '../components/CredentialsSection';
import HostControls, { unavailableReason } from '../components/HostControls';
import EventHistory from '../components/EventHistory';
import PageHeader from '../components/PageHeader';
import UptimeSummary from '../components/UptimeSummary';
import ThresholdFields, {
  EMPTY_ROW,
  METRIC_KEYS,
  thresholdToDisplay,
  thresholdToWire,
  type ThresholdRows,
} from '../components/ThresholdFields';

const REFRESH_MS = 15000;

const EMPTY_INV: Inv = { services: [], containers: [], cron_jobs: [] };

// The host page is five pages: what it is doing now, what it has been doing,
// what it runs, what it costs to log in to, and how it is configured. The tab
// lives in the URL so a link to a machine's metrics stays a link to its metrics.
const HOST_TABS = ['overview', 'metrics', 'inventory', 'credentials', 'settings'] as const;
type HostTab = (typeof HOST_TABS)[number];

export default function HostInventory() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const [params, setParams] = useSearchParams();
  const admin = user?.role === 'admin';
  const [inv, setInv] = useState<Inv | null>(null);
  const [host, setHost] = useState<Host | null>(null);
  const [metricsKey, setMetricsKey] = useState(0);
  const [clearing, setClearing] = useState(false);

  // What a host runs is admin-only, so a basic account gets an empty inventory
  // rather than a rejected request. Nothing here may block the page: this used
  // to hold every tab behind a fetch that a non-admin can no longer make.
  const load = useCallback(() => {
    if (admin) {
      api.get<Inv>(`/api/hosts/${id}/inventory`).then(setInv).catch(() => setInv(EMPTY_INV));
    } else {
      setInv(EMPTY_INV);
    }
    api.get<Host[]>('/api/hosts').then((hs) => setHost((hs ?? []).find((h) => h.id === id) ?? null));
  }, [id, admin]);

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
    navigate(`/services/new?${q}`);
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

  const requested = params.get('tab') as HostTab | null;
  let tab: HostTab = 'overview';
  const adminOnlyTab = requested === 'settings' || requested === 'inventory';
  if (requested && HOST_TABS.includes(requested) && (!adminOnlyTab || admin)) {
    tab = requested;
  }
  const setTab = (next: HostTab) => setParams({ tab: next }, { replace: true });

  const inventoryCount = inv.services.length + inv.containers.length + inv.cron_jobs.length;
  const tabs: { key: HostTab; label: string; count?: number }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'metrics', label: 'Metrics' },
  ];
  if (admin) {
    tabs.push({ key: 'inventory', label: 'Inventory', count: inventoryCount });
  }
  tabs.push({ key: 'credentials', label: 'Credentials' });
  if (admin) {
    tabs.push({ key: 'settings', label: 'Settings' });
  }

  return (
    <div>
      <PageHeader
        title={host?.name ?? 'Host'}
        icon={<EntityIcon url={host?.icon_url} name={host?.name ?? 'Host'} size={36} />}
        badges={host && <Pill tone={hostTone(host.status)}>{host.status}</Pill>}
        subtitle={hostSubtitle(host, admin)}
        meta={id && <UptimeSummary path={`/api/hosts/${id}/uptime`} />}
      />

      <div className="mt-5">
        <Tabs tabs={tabs} active={tab} onChange={setTab} label="Host sections" />
      </div>

      {/* One stack per tab, so the gap between two sections is the same gap
          everywhere instead of whatever each section brought with it. */}
      <div className="mt-6 space-y-6">
        {tab === 'overview' && (
          <>
            {host?.thumbnail_url && (
              <img
                src={host.thumbnail_url}
                alt=""
                className="w-full max-w-lg rounded-card border border-hairline object-cover"
              />
            )}
            {host && <RightNow host={host} />}
            {id && host && admin && <HostControls hostId={id} host={host} />}
            {host && admin && <AgentSection host={host} onChanged={load} />}
            {id && <EventHistory path={`/api/hosts/${id}/events`} />}
          </>
        )}

        {tab === 'metrics' && id && (
          <HostMetrics
            key={metricsKey}
            path={`/api/hosts/${id}/metrics`}
            action={
              admin && (
                <Button variant="secondary" onClick={() => setClearing(true)}>
                  Clear metrics
                </Button>
              )
            }
          />
        )}

        {tab === 'inventory' && (
          <>
            <InventorySection
              title="Systemd units"
              detailLabel="Sub-state"
              items={inv.services}
              onCreate={createFrom}
              control={controlFor('service')}
              hostName={hostLabel}
              emptyDescription="This machine reports no systemd units, or its agent cannot read them."
            />
            <InventorySection
              title="Containers"
              detailLabel="Image"
              items={inv.containers}
              onCreate={createFrom}
              control={controlFor('container')}
              hostName={hostLabel}
              emptyDescription="No container runtime is reporting on this machine."
            />
            <InventorySection
              title="Cron jobs"
              detailLabel="Schedule"
              items={inv.cron_jobs}
              onCreate={createFrom}
              hostName={hostLabel}
              emptyDescription="No scheduled job is set up for any user on this machine."
            />
          </>
        )}

        {tab === 'credentials' && id && <CredentialsSection hostId={id} canManage={admin} />}

        {tab === 'settings' && admin && (
          <>
            {id && host && <HostIdentity host={host} onSaved={load} />}

            {id && host && (
              <Section title="Location">
                <Card className="p-4">
                  <MapPicker
                    hostId={id}
                    location={host.physical_location ?? ''}
                    latitude={host.latitude}
                    longitude={host.longitude}
                    pinColor={host.pin_color}
                    onSaved={load}
                  />
                </Card>
              </Section>
            )}

            {id && (
              <div className="grid gap-6 md:grid-cols-2">
                <Section title="Icon">
                  <Card className="p-4">
                    <IconUploader
                      url={host?.icon_url}
                      name={host?.name ?? 'Host'}
                      path={`/api/admin/hosts/${id}/icon`}
                      onChange={load}
                    />
                  </Card>
                </Section>
                <Section title="Thumbnail">
                  <Card className="p-4">
                    <ThumbnailUploader
                      url={host?.thumbnail_url}
                      path={`/api/admin/hosts/${id}/thumbnail`}
                      onChange={load}
                    />
                  </Card>
                </Section>
              </div>
            )}

            {id && <HostThresholds hostId={id} />}

            {id && host && !host.is_server && <DeleteHost hostId={id} host={host} />}
          </>
        )}
      </div>

      {clearing && (
        <ConfirmModal
          title="Clear stored metrics?"
          body={`Every metric sample, container stat and process average kept for ${hostLabel} is deleted. Current status and inventory stay. This cannot be undone.`}
          confirmLabel="Clear metrics"
          onConfirm={clearMetrics}
          onClose={() => setClearing(false)}
        />
      )}
    </div>
  );
}

// hostSubtitle is the one line under the name: what the machine is and where it
// answers from. Anything the host has not reported is left out rather than
// printed as an em dash, so the line stays short on a host that just enrolled.
// The location is a link to the tab that owns it, which is where an operator goes
// after reading an address that is wrong.
function hostSubtitle(host: Host | null, canEditLocation: boolean): ReactNode {
  if (!host) {
    return undefined;
  }
  const parts: ReactNode[] = [];
  if (host.description) {
    parts.push(<span key="description" className="block text-content">{host.description}</span>);
  }
  if (host.os) {
    parts.push(host.os);
  }
  if (host.ip_address) {
    parts.push(host.ip_address);
  }
  if (host.physical_location) {
    if (canEditLocation) {
      parts.push(
        <Link
          key="location"
          to={`/hosts/${host.id}?tab=settings`}
          className="text-link underline underline-offset-2 hover:text-link-hover"
        >
          {host.physical_location}
        </Link>,
      );
    } else {
      parts.push(host.physical_location);
    }
  }
  if (parts.length === 0) {
    return undefined;
  }
  return (
    <>
      {parts.map((part, i) => (
        <span key={i}>
          {i > 0 && ' · '}
          {part}
        </span>
      ))}
    </>
  );
}

// RightNow answers the question the overview is for: what is this machine doing
// at this moment. It used to open on two admin-only cards of controls, so a
// non-admin's overview was blank and an admin's led with a shutdown button.
function RightNow({ host }: { host: Host }) {
  const m = host.metrics;
  const pct = (used: number, total: number) => {
    if (total <= 0) {
      return 0;
    }
    return (used / total) * 100;
  };

  let lastSeen = 'never';
  if (host.last_seen_at) {
    lastSeen = new Date(host.last_seen_at).toLocaleString();
  }

  return (
    <Section title="Right now">
      <Card className="p-4">
        {m ? (
          <div className="grid gap-4 sm:grid-cols-3">
            <MetricBar label="CPU" pct={m.cpu_pct} detail={`${m.cpu_pct.toFixed(0)}%`} />
            <MetricBar
              label="Memory"
              pct={pct(m.mem_used, m.mem_total)}
              detail={`${fmtBytes(m.mem_used)} / ${fmtBytes(m.mem_total)}`}
            />
            <MetricBar
              label="Disk (root)"
              pct={pct(m.disk_used, m.disk_total)}
              detail={`${fmtBytes(m.disk_used)} / ${fmtBytes(m.disk_total)}`}
            />
          </div>
        ) : (
          <p className="text-sm text-muted">
            This host has not pushed a metric sample yet. It appears here within ~15s of the agent
            starting.
          </p>
        )}
        <div className="mt-5 border-t border-hairline pt-4">
          <Facts
            items={[
              { label: 'State', value: host.status },
              { label: 'Last seen', value: lastSeen },
              { label: 'Agent', value: host.agent_version || 'never reported' },
              { label: 'Address', value: host.ip_address || 'not reported' },
            ]}
          />
        </div>
      </Card>
    </Section>
  );
}

function AgentSection({ host, onChanged }: { host: Host; onChanged: () => void }) {
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [installing, setInstalling] = useState(false);

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
    <Section
      title="Agent updates"
      action={<Pill tone={UPDATE_TONE[host.update_state]}>{UPDATE_LABEL[host.update_state]}</Pill>}
    >
      <Card className="p-4">
        {stateNote && <p className="mb-4 text-xs text-muted">{stateNote}</p>}
        <div className="flex flex-wrap items-end gap-3">
          <Field label="Auto-update">
            <Select
              value={host.auto_update}
              onChange={(e) => setPolicy(e.target.value as AutoUpdatePolicy)}
              disabled={busy}
            >
              {(['default', 'on', 'off'] as const).map((p) => (
                <option key={p} value={p}>
                  {POLICY_LABEL[p]}
                </option>
              ))}
            </Select>
          </Field>
          {hostRowActions(host).update && (
            <Button variant="secondary" onClick={updateNow} disabled={busy}>
              Update now
            </Button>
          )}
          <Button variant="secondary" onClick={() => setInstalling(true)}>
            Install command
          </Button>
        </div>
        <ErrorText>{error}</ErrorText>
      </Card>
      {installing && <AgentInstallModal host={host} onClose={() => setInstalling(false)} />}
    </Section>
  );
}

// HostIdentity is what a machine is called and what it is for. The name a host
// was enrolled under is a label somebody typed once, the server's own row starts
// on a default nobody chose, and what a box is actually for is not derivable
// from anything it reports.
function HostIdentity({ host, onSaved }: { host: Host; onSaved: () => void }) {
  const [name, setName] = useState(host.name);
  const [description, setDescription] = useState(host.description ?? '');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    setName(host.name);
    setDescription(host.description ?? '');
  }, [host.name, host.description]);

  const save = async () => {
    setError('');
    setBusy(true);
    try {
      await api.put(`/api/admin/hosts/${host.id}/identity`, {
        name: name.trim(),
        description: description.trim(),
      });
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not save this host');
    } finally {
      setBusy(false);
    }
  };

  const dirty = name.trim() !== '' && (name.trim() !== host.name || description.trim() !== (host.description ?? ''));

  return (
    <Section title="Name and description">
      <Card className="p-4">
        <Form onSubmit={save}>
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Name">
              <Input value={name} onChange={(e) => setName(e.target.value)} />
            </Field>
            <Field label="Description" hint="Optional. What this machine is for.">
              <Input value={description} onChange={(e) => setDescription(e.target.value)} />
            </Field>
          </div>
          <div className="mt-4 flex items-center gap-3">
            <Button type="submit" disabled={!dirty || busy}>
              Save
            </Button>
          </div>
          <ErrorText>{error}</ErrorText>
        </Form>
      </Card>
    </Section>
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
    <Section
      title="Alert thresholds"
    >
      <Card className="p-4">
        <Form onSubmit={save}>
          <ThresholdFields
            value={rows}
            onChange={(metric, row) => setRows((r) => ({ ...r, [metric]: row }))}
          />
          <div className="mt-4 flex items-center gap-3">
            <Button type="submit">Save</Button>
            <Button variant="secondary" onClick={reset}>
              Reset to fleet default
            </Button>
            {saved && <span className="text-sm text-muted">Saved.</span>}
          </div>
          <ErrorText>{error}</ErrorText>
        </Form>
      </Card>
    </Section>
  );
}

// RowControls queues one action for a single unit or container. It reports its
// own failure inline: a button that silently does nothing is worse than none.
// Stop and restart ask first — they take whatever is using the unit down with
// them, and these rows sit close enough together to hit the wrong one.
const VERBS = [
  { verb: 'start', label: 'Start' },
  { verb: 'stop', label: 'Stop' },
  { verb: 'restart', label: 'Restart' },
];

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
    <div className="flex items-center justify-end gap-1.5">
      {failed && (
        <span className="max-w-40 truncate text-xs text-down" title={failed}>
          {failed}
        </span>
      )}
      {VERBS.map((v) => (
        <button
          key={v.verb}
          type="button"
          disabled={busy}
          onClick={() => click(v.verb)}
          aria-label={`${v.label} ${target}`}
          className="rounded-button px-2 py-1 text-xs font-medium text-muted transition-colors hover:bg-surface-2 hover:text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-link disabled:opacity-50"
        >
          {v.label}
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

// DeleteHost removes the host and everything the schema hangs off it, and is
// where removing the agent from the machine lives too — both are ways of ending
// this host, and neither belongs one click away in a list. The confirm names
// what goes with it, because credentials and history cascade and a catalogued
// service does not: tools keep a host_id that no longer resolves.
function DeleteHost({ hostId, host }: { hostId: string; host: Host }) {
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState(false);
  const [uninstalling, setUninstalling] = useState(false);
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
    <Section title="Remove this host">
      <Card className="border-down-line p-4">
        <p className="text-xs text-muted">
          Removing the agent stops {host.name} reporting but leaves it here with its history. Deleting
          the host removes it, its stored credentials, metrics, events and command history — the agent
          on the machine keeps running until you uninstall it, and any service pinned to this host
          keeps a reference that no longer resolves.
        </p>
        <div className="mt-4 flex flex-wrap gap-3">
          <Button variant="secondary" onClick={() => setUninstalling(true)}>
            Remove agent over SSH
          </Button>
          <Button variant="danger" onClick={() => setConfirming(true)}>
            Delete this host
          </Button>
        </div>
        <ErrorText>{error}</ErrorText>

        {uninstalling && (
          <SSHDeployModal
            hostId={hostId}
            hostName={host.name}
            mode="uninstall"
            onClose={() => setUninstalling(false)}
            onDone={() => undefined}
          />
        )}

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
    </Section>
  );
}

// InventorySection is one of the three lists of what a machine runs. A table
// rather than a stack of rows: the state word and the row's buttons mean nothing
// without a column naming them, and the same list of units used to read as
// "cups.service / inactive/dead" in muted monospace.
function InventorySection({
  title,
  detailLabel,
  items,
  onCreate,
  control,
  hostName,
  emptyDescription,
}: {
  title: string;
  detailLabel: string;
  items: InventoryItem[];
  onCreate: (i: InventoryItem) => void;
  control?: ((verb: string, target: string) => Promise<void>) | null;
  hostName: string;
  emptyDescription: string;
}) {
  const [search, setSearch] = useState('');
  const shown = items.filter((it) => matchesQuery(search, it.name, it.detail, it.state));
  const hasState = items.some((it) => Boolean(it.state));

  let searchBox = null;
  if (items.length > 0) {
    searchBox = (
      <SearchBar
        value={search}
        onChange={setSearch}
        placeholder={`Search ${title.toLowerCase()}…`}
        shortcut={false}
        className="w-56"
      />
    );
  }

  let body = (
    <Table
      head={
        <>
          <Th>Name</Th>
          {hasState && <Th>State</Th>}
          <Th>{detailLabel}</Th>
          <Th className="text-right">Monitoring</Th>
        </>
      }
    >
      {shown.map((it) => (
        <Tr key={it.source_ref}>
          <Td className="max-w-xs truncate font-medium text-content" title={it.name}>
            {it.name}
          </Td>
          {hasState && (
            <Td>{it.state && <Pill tone={unitStateTone(it.state)}>{it.state}</Pill>}</Td>
          )}
          <Td className="max-w-xs truncate font-mono text-xs text-muted" title={it.detail}>
            {it.detail || '—'}
          </Td>
          <Td className="text-right">
            <div className="flex items-center justify-end gap-2">
              {control && <RowControls target={it.source_ref} hostName={hostName} run={control} />}
              {it.linked ? (
                <Pill tone="up">monitored</Pill>
              ) : (
                <Button variant="secondary" onClick={() => onCreate(it)}>
                  Add for monitoring
                </Button>
              )}
            </div>
          </Td>
        </Tr>
      ))}
    </Table>
  );
  if (items.length === 0) {
    body = <EmptyState title="None reported" description={emptyDescription} />;
  } else if (shown.length === 0) {
    body = (
      <EmptyState
        title="Nothing here matches the search"
        description={`No ${title.toLowerCase()} on this host matches “${search.trim()}”.`}
      />
    );
  }

  return (
    <Section title={title} count={shown.length} action={searchBox}>
      {body}
    </Section>
  );
}
