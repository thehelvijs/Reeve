import { useCallback, useEffect, useState } from 'react';
import { api, type Host, type HostCommand } from '../api';
import { Button, Card, ErrorText, Form, Pill, Section, Table, Td, Th, Tr } from './ui';
import Modal from './Modal';

const REFRESH_MS = 5000;

export const ACTION_LABEL: Record<string, string> = {
  reboot: 'Restart',
  poweroff: 'Shut down',
  service_start: 'Start',
  service_stop: 'Stop',
  service_restart: 'Restart',
  container_start: 'Start',
  container_stop: 'Stop',
  container_restart: 'Restart',
};

const STATUS_TONE: Record<HostCommand['status'], 'up' | 'down' | 'muted'> = {
  pending: 'muted',
  sent: 'muted',
  done: 'up',
  failed: 'down',
  expired: 'down',
};

// unavailableReason explains a disabled control rather than leaving a dead
// button. The two causes are indistinguishable in the data — an agent that
// predates the feature and one whose host opted out both simply never said so.
export function unavailableReason(host: Host): string {
  if (host.status !== 'online') {
    return 'This host is offline. Commands are collected on the agent’s next push, so there is nothing to send them to.';
  }
  if (!host.control_enabled) {
    return 'This host’s agent has not reported control support — it predates the feature, or the machine runs it with REEVE_ALLOW_CONTROL=false.';
  }
  return '';
}

export default function HostControls({ hostId, host }: { hostId: string; host: Host }) {
  const [commands, setCommands] = useState<HostCommand[]>([]);
  const [confirming, setConfirming] = useState<'reboot' | 'poweroff' | null>(null);
  const [error, setError] = useState('');
  const [shown, setShown] = useState<HostCommand | null>(null);

  const load = useCallback(() => {
    api.get<HostCommand[]>(`/api/admin/hosts/${hostId}/commands`).then((c) => setCommands(c ?? []));
  }, [hostId]);

  useEffect(() => {
    load();
    const t = setInterval(load, REFRESH_MS);
    return () => clearInterval(t);
  }, [load]);

  const send = async (action: string) => {
    setError('');
    try {
      await api.post(`/api/admin/hosts/${hostId}/commands`, { action, target: '' });
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not queue the command');
    }
    setConfirming(null);
  };

  const reason = unavailableReason(host);

  return (
    <Section
      title="Controls"
    >
      <Card className="p-4">
        {reason && <p className="mb-4 text-xs text-muted">{reason}</p>}

        <div className="flex flex-wrap gap-3">
          <Button variant="secondary" disabled={reason !== ''} onClick={() => setConfirming('reboot')}>
            Restart host
          </Button>
          <Button variant="danger" disabled={reason !== ''} onClick={() => setConfirming('poweroff')}>
            Shut down host
          </Button>
        </div>
        <ErrorText>{error}</ErrorText>

        {commands.length > 0 && (
          <div className="mt-5">
            <p className="mb-2 text-eyebrow font-semibold uppercase text-muted">Recent commands</p>
            <Table
              head={
                <>
                  <Th>Command</Th>
                  <Th>Asked by</Th>
                  <Th>When</Th>
                  <Th>Result</Th>
                </>
              }
            >
              {commands.map((c) => (
                <Tr key={c.id}>
                  <Td>
                    {/* The name opens what the command printed. A table row
                        cannot itself be a button without breaking its cells. */}
                    <button
                      type="button"
                      onClick={() => setShown(c)}
                      className="text-left font-medium text-link transition-colors hover:text-link-hover hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-link"
                    >
                      {ACTION_LABEL[c.action] ?? c.action}
                      {c.target && <span className="ml-1 font-mono text-xs text-muted">{c.target}</span>}
                    </button>
                  </Td>
                  <Td className="truncate text-muted">{c.requested_by_name}</Td>
                  <Td className="whitespace-nowrap text-muted">{new Date(c.requested_at).toLocaleString()}</Td>
                  <Td>
                    <Pill tone={STATUS_TONE[c.status]}>{c.status}</Pill>
                  </Td>
                </Tr>
              ))}
            </Table>
          </div>
        )}
      </Card>

      {confirming && (
        <Modal
          title={confirming === 'reboot' ? 'Restart this host?' : 'Shut down this host?'}
          onClose={() => setConfirming(null)}
        >
          <Form onSubmit={() => send(confirming)}>
            <p className="text-sm text-muted">
              {confirming === 'reboot'
                ? `${host.name} will reboot as soon as its agent picks this up. Everything running on it goes down until it comes back.`
                : `${host.name} will power off as soon as its agent picks this up. Nothing here can turn it back on.`}
            </p>
            <div className="mt-5 flex justify-end gap-2">
              <Button type="button" variant="secondary" onClick={() => setConfirming(null)}>
                Cancel
              </Button>
              <Button type="submit" variant="danger">
                {confirming === 'reboot' ? 'Restart' : 'Shut down'}
              </Button>
            </div>
          </Form>
        </Modal>
      )}

      {shown && (
        <Modal
          title={`${ACTION_LABEL[shown.action] ?? shown.action} ${shown.target}`.trim()}
          onClose={() => setShown(null)}
        >
          <p className="text-xs text-muted">
            Requested by {shown.requested_by_name} · {new Date(shown.requested_at).toLocaleString()} ·{' '}
            {shown.status}
          </p>
          <pre className="mt-3 max-h-64 overflow-auto whitespace-pre-wrap rounded-button border border-hairline bg-surface-2 p-3 font-mono text-xs text-content">
            {shown.output || 'No output.'}
          </pre>
          <div className="mt-4 flex justify-end">
            <Button onClick={() => setShown(null)}>Close</Button>
          </div>
        </Modal>
      )}
    </Section>
  );
}
