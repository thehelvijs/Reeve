import { useState, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { api, type AgentUpdateRollup, type Host } from '../api';
import { useAuth } from '../auth';
import { Button, Card, ErrorText, Field, Input, Pill } from '../components/ui';
import PageHeader from '../components/PageHeader';
import Modal from '../components/Modal';
import EmptyState from '../components/EmptyState';
import EntityIcon from '../components/EntityIcon';
import Chevron from '../components/Chevron';
import { ListSkeleton } from '../components/Skeleton';
import SSHDeployModal from '../components/SSHDeployModal';
import { useResource } from '../lib/cache';
import { UPDATE_LABEL, UPDATE_TONE, showsVersionPill } from '../lib/agentUpdate';

export default function Hosts() {
  const { user } = useAuth();
  const [adding, setAdding] = useState(false);
  const [tutorial, setTutorial] = useState(false);
  const [deploy, setDeploy] = useState<{ host: Host; mode: 'install' | 'uninstall' } | null>(null);
  const { data, loading, refresh } = useResource<Host[]>(
    '/api/v1/hosts',
    () => api.get<Host[]>('/api/v1/hosts'),
    15000,
  );
  const hosts = data ?? [];

  const tone = (s: Host['status']) => (s === 'online' ? 'up' : s === 'offline' ? 'down' : 'muted');

  return (
    <div>
      <PageHeader
        title="Hosts"
        subtitle="Machines running the agent."
        action={
          user?.role === 'admin' && (
            <div className="flex gap-2">
              <Button variant="secondary" onClick={() => setTutorial(true)}>
                How to add a host
              </Button>
              <Button onClick={() => setAdding(true)}>Add host</Button>
            </div>
          )
        }
      />

      {user?.role === 'admin' && <AgentRollup />}

      {loading ? (
        <div className="mt-6">
          <ListSkeleton />
        </div>
      ) : hosts.length === 0 ? (
        <div className="mt-6">
          <EmptyState
            title="No hosts yet"
            description="Enroll a machine to start collecting its status and metrics through the agent."
            action={
              user?.role === 'admin' && <Button onClick={() => setAdding(true)}>Add host</Button>
            }
          />
        </div>
      ) : (
        <div className="mt-6 divide-y divide-hairline overflow-hidden rounded-card border border-hairline">
          {hosts.map((h) => (
            <div key={h.id} className="flex items-center transition-colors hover:bg-surface-2">
              <Link to={`/hosts/${h.id}`} className="flex min-w-0 flex-1 items-center gap-3 px-4 py-3">
                <EntityIcon url={h.icon_url} name={h.name} size={36} />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-content">{h.name}</p>
                  <p className="truncate text-xs text-muted">
                    {h.os}
                    {h.agent_version ? ` · agent ${h.agent_version}` : ''}
                    {h.last_seen_at ? ` · seen ${new Date(h.last_seen_at).toLocaleString()}` : ''}
                  </p>
                </div>
                {showsVersionPill(h.update_state) && (
                  <Pill tone={UPDATE_TONE[h.update_state]}>{UPDATE_LABEL[h.update_state]}</Pill>
                )}
                <Pill tone={tone(h.status)}>{h.status}</Pill>
              </Link>
              {user?.role === 'admin' && (
                <div className="flex shrink-0 gap-2 pr-2">
                  <Button variant="secondary" onClick={() => setDeploy({ host: h, mode: 'install' })}>
                    Install over SSH
                  </Button>
                  <Button variant="danger" onClick={() => setDeploy({ host: h, mode: 'uninstall' })}>
                    Remove agent
                  </Button>
                </div>
              )}
              <Link to={`/hosts/${h.id}`} className="pr-4">
                <Chevron />
              </Link>
            </div>
          ))}
        </div>
      )}

      {tutorial && <TutorialModal onClose={() => setTutorial(false)} onAdd={() => { setTutorial(false); setAdding(true); }} />}
      {adding && (
        <EnrollModal
          onClose={() => setAdding(false)}
          onSaved={refresh}
          onSSH={(host) => {
            setAdding(false);
            setDeploy({ host, mode: 'install' });
          }}
        />
      )}
      {deploy && (
        <SSHDeployModal
          hostId={deploy.host.id}
          hostName={deploy.host.name}
          mode={deploy.mode}
          onClose={() => setDeploy(null)}
          onDone={refresh}
        />
      )}
    </div>
  );
}

function AgentRollup() {
  const { data, refresh } = useResource<AgentUpdateRollup>(
    '/api/v1/admin/agent-updates',
    () => api.get<AgentUpdateRollup>('/api/v1/admin/agent-updates'),
    15000,
  );
  const [error, setError] = useState('');
  if (!data) {
    return null;
  }

  const resume = async () => {
    setError('');
    try {
      await api.post('/api/v1/admin/agent-updates/resume', {});
      refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not resume the rollout');
    }
  };

  const parts: string[] = [`server on ${data.server_version}`];
  for (const state of ['up_to_date', 'outdated', 'updating', 'stalled', 'disabled', 'unknown'] as const) {
    const n = data.counts[state];
    if (n > 0) {
      parts.push(`${n} ${UPDATE_LABEL[state]}`);
    }
  }

  return (
    <Card className="mt-4 px-4 py-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-xs text-muted">{parts.join(' · ')}</p>
        {data.paused && (
          <Button variant="secondary" onClick={resume}>
            Resume rollout
          </Button>
        )}
      </div>
      {data.paused && (
        <p className="mt-2 text-xs text-muted">
          Rollout paused: {data.stalled.join(', ')} did not come back on the new agent. No other host
          updates until this is cleared.
        </p>
      )}
      <ErrorText>{error}</ErrorText>
    </Card>
  );
}

interface EnrollResult {
  host: Host;
  enroll_token: string;
  install_command: string;
  run_command: string;
}

function EnrollModal({
  onClose,
  onSaved,
  onSSH,
}: {
  onClose: () => void;
  onSaved: () => void;
  onSSH: (host: Host) => void;
}) {
  const [name, setName] = useState('');
  const [location, setLocation] = useState('');
  const [result, setResult] = useState<EnrollResult | null>(null);
  const [error, setError] = useState('');

  const create = async () => {
    setError('');
    try {
      const res = await api.post<EnrollResult>('/api/v1/admin/hosts', {
        name,
        physical_location: location,
      });
      setResult(res);
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed');
    }
  };

  return (
    <Modal title="Add host" onClose={onClose}>
        {!result ? (
          <div className="mt-4 space-y-3">
            <Field label="Name">
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="db-server-1" />
            </Field>
            <Field label="Physical location">
              <Input value={location} onChange={(e) => setLocation(e.target.value)} placeholder="rack 3" />
            </Field>
            <ErrorText>{error}</ErrorText>
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="secondary" onClick={onClose}>
                Cancel
              </Button>
              <Button onClick={create} disabled={!name.trim()}>
                Create
              </Button>
            </div>
          </div>
        ) : (
          <div className="mt-4 space-y-3">
            <p className="text-sm text-content">
              Host created. Let the server install the agent over SSH, or run one of these on the machine as root — the
              enrollment token is shown only once. It turns online here within ~15s once the agent starts.
            </p>
            <CmdBlock label="Install (systemd)" cmd={result.install_command} />
            <CmdBlock label="Or run in Docker" cmd={result.run_command} />
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={onClose}>
                Done
              </Button>
              <Button onClick={() => onSSH(result.host)}>Install over SSH</Button>
            </div>
          </div>
        )}
    </Modal>
  );
}

function CmdBlock({ label, cmd }: { label: string; cmd: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    await navigator.clipboard.writeText(cmd);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  return (
    <div>
      <div className="mb-1 flex items-center justify-between">
        <span className="text-xs font-medium text-muted">{label}</span>
        <button type="button" onClick={copy} className="text-xs text-muted transition-colors hover:text-content">
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>
      <pre className="overflow-x-auto rounded-button border border-hairline bg-surface-2 px-3 py-2 font-mono text-xs text-content">
        {cmd}
      </pre>
    </div>
  );
}

function TutorialModal({ onClose, onAdd }: { onClose: () => void; onAdd: () => void }) {
  return (
    <Modal title="How to add a host" onClose={onClose}>
      <ol className="mt-4 space-y-4">
        <Step n={1} title="Create the host here">
          Click “Add host” and give it a name. You get a one-line install command with a unique enrollment token.
        </Step>
        <Step n={2} title="Install the agent, pushed or pulled">
          Either let the server do it — “Install over SSH” on the host row, giving it SSH credentials once — or copy the
          one-line command and run it on the machine as root. Both run the same installer; re-run anytime to upgrade.
        </Step>
        <Step n={3} title="It appears automatically">
          The agent pushes status, inventory, and metrics every ~15s over an outbound connection. The host turns
          online here within a few seconds — no inbound ports to open.
        </Step>
      </ol>
      <div className="mt-6 flex justify-end gap-2">
        <Button variant="secondary" onClick={onClose}>
          Close
        </Button>
        <Button onClick={onAdd}>Add host</Button>
      </div>
    </Modal>
  );
}

function Step({ n, title, children }: { n: number; title: string; children: ReactNode }) {
  return (
    <li className="flex gap-3">
      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-hairline text-xs font-medium text-content">
        {n}
      </span>
      <div>
        <p className="text-sm font-medium text-content">{title}</p>
        <p className="mt-0.5 text-sm text-muted">{children}</p>
      </div>
    </li>
  );
}
