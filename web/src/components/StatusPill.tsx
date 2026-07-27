import type { ToolStatus } from '../api';
import { Pill } from './ui';
import { TOOL_LABEL, TOOL_TONE } from '../lib/statusTone';

export default function StatusPill({ status }: { status: ToolStatus }) {
  const label = TOOL_LABEL[status] ?? TOOL_LABEL.unknown;
  const tone = TOOL_TONE[status] ?? TOOL_TONE.unknown;
  return <Pill tone={tone}>{label}</Pill>;
}
