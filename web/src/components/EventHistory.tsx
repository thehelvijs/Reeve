import { useEffect, useState } from 'react';
import { api, type AlertEvent } from '../api';
import { Card, Pill } from '../components/ui';

export default function EventHistory({ path }: { path: string }) {
  const [events, setEvents] = useState<AlertEvent[]>([]);

  useEffect(() => {
    const load = () => api.get<AlertEvent[]>(path).then((e) => setEvents(e ?? [])).catch(() => setEvents([]));
    load();
    const poll = setInterval(load, 15000);
    return () => clearInterval(poll);
  }, [path]);

  return (
    <div className="mt-6">
      <h2 className="text-sm font-medium text-content">Event history</h2>
      <Card className="mt-2 max-h-80 divide-y divide-hairline overflow-y-auto">
        {events.length === 0 && <p className="px-4 py-3 text-sm text-muted">No events.</p>}
        {events.map((e) => (
          <div key={e.id} className="flex items-center justify-between px-4 py-2.5 text-sm">
            <div className="flex items-center gap-2">
              <Pill tone={e.resolved_at ? 'up' : 'down'}>{e.type.replace('_', ' ')}</Pill>
              <span className="truncate text-content">{e.message}</span>
            </div>
            <span className="shrink-0 text-xs text-muted">
              {e.resolved_at ? 'resolved' : 'firing'} · {new Date(e.fired_at).toLocaleString()}
            </span>
          </div>
        ))}
      </Card>
    </div>
  );
}
