import { useState } from 'react';
import { api, type AlertEvent } from '../api';
import { useResource } from '../lib/cache';
import { Card, Pill } from '../components/ui';
import SearchBar from './SearchBar';
import { matchesQuery } from '../lib/search';

export default function EventHistory({ path }: { path: string }) {
  const { data } = useResource<AlertEvent[]>(path, () => api.get<AlertEvent[]>(path), 15000);
  const events = data ?? [];
  const [search, setSearch] = useState('');

  const shown = events.filter((e) => matchesQuery(search, e.type.replace('_', ' '), e.message));
  let empty = 'No events.';
  if (search.trim()) {
    empty = 'No event matches the search.';
  }

  return (
    <div className="mt-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-sm font-medium text-content">Event history</h2>
        {events.length > 0 && (
          <SearchBar
            value={search}
            onChange={setSearch}
            placeholder="Search events…"
            shortcut={false}
            className="w-56"
          />
        )}
      </div>
      <Card className="mt-2 max-h-80 divide-y divide-hairline overflow-y-auto">
        {shown.length === 0 && <p className="px-4 py-3 text-sm text-muted">{empty}</p>}
        {shown.map((e) => (
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
