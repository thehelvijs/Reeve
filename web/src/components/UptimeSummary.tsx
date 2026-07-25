import { useEffect, useState } from 'react';
import { api, type UptimeSummary as Summary } from '../api';

export default function UptimeSummary({ path }: { path: string }) {
  const [day, setDay] = useState<Summary | null>(null);
  const [week, setWeek] = useState<Summary | null>(null);

  useEffect(() => {
    api.get<Summary>(`${path}?range=24h`).then(setDay).catch(() => setDay(null));
    api.get<Summary>(`${path}?range=7d`).then(setWeek).catch(() => setWeek(null));
  }, [path]);

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
