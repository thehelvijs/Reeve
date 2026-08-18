import { useState } from 'react';
import { api, type Host } from '../api';
import { Button, ErrorText } from './ui';
import Modal from './Modal';

// AgentInstallModal hands over the commands that put an agent on a machine
// Reeve already knows about — a host somebody created earlier, or the one the
// server itself runs on. The token is minted on demand rather than on open,
// because minting it revokes whatever the machine is using now.
export default function AgentInstallModal({ host, onClose }: { host: Host; onClose: () => void }) {
  const [cmds, setCmds] = useState<{ install_command: string } | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const issue = async () => {
    setError('');
    setBusy(true);
    try {
      setCmds(await api.post(`/api/admin/hosts/${host.id}/enroll-token`, {}));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not issue an enrollment token');
    } finally {
      setBusy(false);
    }
  };

  let warning =
    'This mints a fresh enrollment token for this host. Run one of the commands on the machine as root and it reports in within ~15s.';
  if (host.agent_version !== '') {
    warning =
      'This host already runs an agent. A fresh token revokes the one it is using, so it stops reporting until you run the new command on it.';
  }

  return (
    <Modal title={`Install the agent on ${host.name}`} onClose={onClose} onSubmit={cmds ? undefined : issue}>
      {!cmds && (
        <div className="mt-4 space-y-3">
          <p className="text-sm text-muted">{warning}</p>
          <ErrorText>{error}</ErrorText>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={issue} disabled={busy}>
              {busy ? 'Issuing…' : 'Show install command'}
            </Button>
          </div>
        </div>
      )}
      {cmds && (
        <div className="mt-4 space-y-3">
          <p className="text-sm text-content">
            Run this on the machine as root — the token is shown only once.
          </p>
          <CmdBlock label="Install (systemd)" cmd={cmds.install_command} />
          <div className="flex justify-end">
            <Button variant="secondary" onClick={onClose}>
              Done
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
}

export function CmdBlock({ label, cmd }: { label: string; cmd: string }) {
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
