import { api, type UptimeSummary as Summary } from '../api';
import { useResource } from '../lib/cache';

export default function UptimeSummary({ path }: { path: string }) {
  const dayKey = `${path}?range=24h`;
  const weekKey = `${path}?range=7d`;
  const { data: day } = useResource<Summary>(dayKey, () => api.get<Summary>(dayKey));
  const { data: week } = useResource<Summary>(weekKey, () => api.get<Summary>(weekKey));

  if (!day && !week) {
    return null;
  }

  return (
    <p className="mt-1 text-xs text-muted">
      {day && <>Uptime {day.uptime_pct}% (24h)</>}
      {day && week && ' · '}
      {week && <>{week.uptime_pct}% (7d)</>}
    </p>
  );
}
