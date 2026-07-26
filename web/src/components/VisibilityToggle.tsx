import { useState } from 'react';
import { api, type Tool } from '../api';
import { Pill } from './ui';
import { toolUpdateBody } from '../lib/tools';

// VisibilityToggle flips one service between public and restricted, which is
// what decides whether anyone who is not signed in sees it on the dashboard.
// A caller who may not edit the tool still reads the state, as a plain pill.
export default function VisibilityToggle({
  tool,
  onChanged,
}: {
  tool: Tool;
  onChanged: (t: Tool) => void;
}) {
  const [busy, setBusy] = useState(false);
  const isPublic = tool.visibility === 'public';
  if (!tool.can_edit) {
    if (isPublic) {
      return null;
    }
    return <Pill tone="down">restricted</Pill>;
  }

  const flip = async (e: React.MouseEvent) => {
    // Lives inside link and card rows, which would otherwise navigate away.
    e.preventDefault();
    e.stopPropagation();
    setBusy(true);
    try {
      const next = isPublic ? 'restricted' : 'public';
      const updated = await api.patch<Tool>(`/api/tools/${tool.id}`, toolUpdateBody(tool, { visibility: next }));
      onChanged(updated);
    } finally {
      setBusy(false);
    }
  };

  let styles = 'bg-green-500/10 text-green-400 border-green-500/20';
  if (!isPublic) {
    styles = 'bg-surface-2 text-muted border-hairline';
  }
  return (
    <button
      type="button"
      onClick={flip}
      disabled={busy}
      aria-pressed={isPublic}
      title={isPublic ? 'Shown to everyone. Click to hide it from the public dashboard.' : 'Hidden from the public dashboard. Click to show it to everyone.'}
      className={`inline-flex shrink-0 items-center rounded-pill border px-2 py-0.5 text-xs transition-colors hover:brightness-125 disabled:opacity-50 ${styles}`}
    >
      {isPublic ? 'public' : 'restricted'}
    </button>
  );
}
