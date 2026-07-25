import type { Tool, ToolStatus } from '../api';

function dotColor(s: ToolStatus): string {
  if (s === 'up') {
    return 'bg-green-400';
  }
  if (s === 'down' || s === 'agent_offline') {
    return 'bg-red-400';
  }
  return 'bg-muted';
}

// Inline health dots for a set of monitored services.
export default function ServiceDots({ tools }: { tools: Tool[] }) {
  if (tools.length === 0) {
    return <span className="text-xs text-muted">No services</span>;
  }
  return (
    <div className="flex flex-wrap gap-x-3 gap-y-1">
      {tools.map((t) => (
        <span key={t.id} className="inline-flex items-center gap-1.5 text-xs text-muted">
          <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${dotColor(t.status)}`} />
          <span className="truncate">{t.name}</span>
        </span>
      ))}
    </div>
  );
}
