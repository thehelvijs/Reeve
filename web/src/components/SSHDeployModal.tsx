import { useState } from 'react';
import { api, ApiError, type SSHTarget } from '../api';
import { Button, ErrorText, Field, Input } from './ui';
import Modal from './Modal';

type Mode = 'install' | 'uninstall';
type Stage = 'form' | 'confirm' | 'running' | 'done';

const EMPTY: SSHTarget = {
  address: '',
  port: 22,
  username: 'root',
  password: '',
  private_key: '',
  passphrase: '',
  sudo_password: '',
  fingerprint: '',
};

// Server-driven agent deployment: the server SSHes into the machine, copies the
// agent, and runs the same installer the curl one-liner runs. Credentials are
// sent for this operation only and never stored.
export default function SSHDeployModal({
  hostId,
  hostName,
  mode,
  onClose,
  onDone,
}: {
  hostId: string;
  hostName: string;
  mode: Mode;
  onClose: () => void;
  onDone: () => void;
}) {
  const [target, setTarget] = useState<SSHTarget>(EMPTY);
  const [auth, setAuth] = useState<'password' | 'key'>('password');
  const [stage, setStage] = useState<Stage>('form');
  const [keyType, setKeyType] = useState('');
  const [output, setOutput] = useState('');
  const [error, setError] = useState('');

  const set = <K extends keyof SSHTarget>(k: K, v: SSHTarget[K]) => setTarget((t) => ({ ...t, [k]: v }));

  const probe = async () => {
    setError('');
    try {
      const res = await api.post<{ fingerprint: string; key_type: string }>('/api/admin/ssh-probe', {
        address: target.address,
        port: Number(target.port),
      });
      set('fingerprint', res.fingerprint);
      setKeyType(res.key_type);
      setStage('confirm');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not reach that host');
    }
  };

  const run = async () => {
    setError('');
    setStage('running');
    const path = mode === 'install' ? 'ssh-install' : 'ssh-uninstall';
    const payload: SSHTarget = {
      ...target,
      port: Number(target.port),
      password: auth === 'password' ? target.password : '',
      private_key: auth === 'key' ? target.private_key : '',
      passphrase: auth === 'key' ? target.passphrase : '',
    };
    try {
      const res = await api.post<{ output: string }>(`/api/admin/hosts/${hostId}/${path}`, payload);
      setOutput(res.output);
      setStage('done');
      onDone();
    } catch (e) {
      if (e instanceof ApiError) {
        const body = e.body as { output?: string } | null;
        setOutput(body?.output ?? '');
        setError(e.message);
      } else {
        setError('failed');
      }
      setStage('confirm');
    }
  };

  const title = mode === 'install' ? `Install agent on ${hostName}` : `Remove agent from ${hostName}`;
  const verb = mode === 'install' ? 'Install' : 'Uninstall';

  const submitAction = () => {
    if (stage === 'form' && target.address.trim() && target.username.trim()) {
      return probe;
    }
    if (stage === 'confirm') {
      return run;
    }
    if (stage === 'done') {
      return onClose;
    }
    return undefined;
  };

  return (
    <Modal title={title} onClose={onClose} onSubmit={submitAction()}>
      {stage === 'form' && (
        <div className="mt-4 space-y-3">
          <p className="text-sm text-muted">
            The server connects to the machine over SSH and runs the {mode === 'install' ? 'installer' : 'uninstaller'}{' '}
            itself. These credentials are used once and never stored.
          </p>
          <div className="grid grid-cols-3 gap-3">
            <div className="col-span-2">
              <Field label="Address">
                <Input
                  value={target.address}
                  placeholder="192.168.1.20"
                  onChange={(e) => set('address', e.target.value)}
                />
              </Field>
            </div>
            <Field label="SSH port">
              <Input type="number" value={target.port} onChange={(e) => set('port', Number(e.target.value))} />
            </Field>
          </div>
          <Field label="Username">
            <Input value={target.username} onChange={(e) => set('username', e.target.value)} />
          </Field>

          <div className="flex gap-4 text-sm text-content">
            {(['password', 'key'] as const).map((m) => (
              <label key={m} className="flex items-center gap-2">
                <input
                  type="radio"
                  className="accent-accent"
                  checked={auth === m}
                  onChange={() => setAuth(m)}
                />
                {m === 'password' ? 'Password' : 'Private key'}
              </label>
            ))}
          </div>

          {auth === 'password' ? (
            <Field label="SSH password">
              <Input
                type="password"
                value={target.password}
                autoComplete="off"
                onChange={(e) => set('password', e.target.value)}
              />
            </Field>
          ) : (
            <>
              <Field label="Private key (PEM)">
                <textarea
                  className="h-28 w-full rounded-button border border-hairline bg-surface-2 px-3 py-2 font-mono text-xs text-content focus:outline-none focus:ring-2 focus:ring-accent"
                  value={target.private_key}
                  placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                  onChange={(e) => set('private_key', e.target.value)}
                />
              </Field>
              <Field label="Key passphrase" hint="Only if the key is encrypted">
                <Input
                  type="password"
                  value={target.passphrase}
                  autoComplete="off"
                  onChange={(e) => set('passphrase', e.target.value)}
                />
              </Field>
            </>
          )}

          {target.username !== 'root' && (
            <Field label="sudo password" hint="Leave empty for passwordless sudo (NOPASSWD)">
              <Input
                type="password"
                value={target.sudo_password}
                autoComplete="off"
                onChange={(e) => set('sudo_password', e.target.value)}
              />
            </Field>
          )}

          <ErrorText>{error}</ErrorText>
          <div className="flex justify-end gap-2 pt-1">
            <Button variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={probe} disabled={!target.address.trim() || !target.username.trim()}>
              Check host key
            </Button>
          </div>
        </div>
      )}

      {(stage === 'confirm' || stage === 'running') && (
        <div className="mt-4 space-y-3">
          <p className="text-sm text-content">Confirm this is the machine you mean:</p>
          <div className="rounded-button border border-hairline bg-surface-2 px-3 py-2">
            <p className="text-xs text-muted">
              {target.address}:{target.port} · {keyType}
            </p>
            <p className="mt-1 select-all break-all font-mono text-xs text-content">{target.fingerprint}</p>
          </div>
          {output && <OutputBlock output={output} />}
          <ErrorText>{error}</ErrorText>
          <div className="flex justify-end gap-2 pt-1">
            <Button variant="secondary" disabled={stage === 'running'} onClick={() => setStage('form')}>
              Back
            </Button>
            <Button variant={mode === 'install' ? 'primary' : 'danger'} disabled={stage === 'running'} onClick={run}>
              {stage === 'running' ? `${verb}ing…` : `${verb} now`}
            </Button>
          </div>
        </div>
      )}

      {stage === 'done' && (
        <div className="mt-4 space-y-3">
          <p className="text-sm text-content">
            {mode === 'install'
              ? 'Agent installed. The host turns online here within about 15 seconds.'
              : 'Agent removed. The host stays in the catalog with its history.'}
          </p>
          <OutputBlock output={output} />
          <div className="flex justify-end">
            <Button onClick={onClose}>Done</Button>
          </div>
        </div>
      )}
    </Modal>
  );
}

function OutputBlock({ output }: { output: string }) {
  return (
    <pre className="max-h-64 overflow-auto rounded-button border border-hairline bg-surface-2 px-3 py-2 font-mono text-[11px] leading-relaxed text-muted">
      {output || '(no output)'}
    </pre>
  );
}
