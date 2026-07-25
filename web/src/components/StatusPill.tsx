import type { ToolStatus } from '../api';
import { Pill } from './ui';

const map: Record<ToolStatus, { tone: 'up' | 'down' | 'muted'; label: string }> = {
  up: { tone: 'up', label: 'up' },
  down: { tone: 'down', label: 'down' },
  agent_offline: { tone: 'down', label: 'agent offline' },
  unknown: { tone: 'muted', label: 'unknown' },
};

export default function StatusPill({ status }: { status: ToolStatus }) {
  const s = map[status] ?? map.unknown;
  return <Pill tone={s.tone}>{s.label}</Pill>;
}
