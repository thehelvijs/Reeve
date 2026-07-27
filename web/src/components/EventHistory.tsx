import { useState } from 'react';
import { api, type AlertEvent } from '../api';
import { useResource } from '../lib/cache';
import { Card, Pill, Section, Table, Td, Th, Tr } from '../components/ui';
import SearchBar from './SearchBar';
import { matchesQuery } from '../lib/search';
import { alertLabel } from '../lib/statusTone';

export default function EventHistory({ path }: { path: string }) {
  const { data } = useResource<AlertEvent[]>(path, () => api.get<AlertEvent[]>(path), 15000);
  const events = data ?? [];
  const [search, setSearch] = useState('');

  const shown = events.filter((e) => matchesQuery(search, alertLabel(e.type), e.message));
  let empty = 'No alert has ever fired here.';
  if (search.trim()) {
    empty = 'No event matches the search.';
  }

  let searchBox = null;
  if (events.length > 0) {
    searchBox = (
      <SearchBar
        value={search}
        onChange={setSearch}
        placeholder="Search events…"
        shortcut={false}
        className="w-56"
      />
    );
  }

  // A resolved alert is history, not good news: `up` green read as "this is
  // fine" on a row saying the host had stopped reporting for 20 minutes.
  return (
    <Section title="Event history" count={events.length} action={searchBox}>
      {shown.length === 0 ? (
        <Card className="p-4">
          <p className="text-sm text-muted">{empty}</p>
        </Card>
      ) : (
        <Table
          head={
            <>
              <Th>Alert</Th>
              <Th>What happened</Th>
              <Th>State</Th>
              <Th>Fired</Th>
            </>
          }
        >
          {shown.map((e) => (
            <Tr key={e.id}>
              <Td className="whitespace-nowrap">
                <Pill tone={e.resolved_at ? 'muted' : 'down'}>{alertLabel(e.type)}</Pill>
              </Td>
              <Td className="text-content">{e.message}</Td>
              <Td className="whitespace-nowrap text-muted">{e.resolved_at ? 'resolved' : 'firing'}</Td>
              <Td className="whitespace-nowrap text-muted">{new Date(e.fired_at).toLocaleString()}</Td>
            </Tr>
          ))}
        </Table>
      )}
    </Section>
  );
}
