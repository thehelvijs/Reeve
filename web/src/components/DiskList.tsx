import { Card } from './ui';
import MetricBar from './MetricBar';
import { fmtBytes } from '../lib/format';
import { sortDisks, type DiskUsage } from '../lib/disks';

// DiskList shows every filesystem the agent reported, fullest first — the point
// of the list is finding the one about to fill up, and that is rarely the root.
export default function DiskList({ disks }: { disks: DiskUsage[] }) {
  if (disks.length === 0) {
    return null;
  }
  return (
    <Card className="mt-4 p-4">
      <p className="mb-3 text-xs font-medium text-muted">
        Filesystems <span className="text-muted">({disks.length})</span>
      </p>
      <div className="max-h-80 space-y-3 overflow-y-auto">
        {sortDisks(disks).map((d) => (
          <MetricBar
            key={d.mount}
            label={`${d.mount} · ${d.device} · ${d.fs_type}`}
            pct={d.total > 0 ? (d.used / d.total) * 100 : 0}
            detail={`${fmtBytes(d.used)} / ${fmtBytes(d.total)}`}
          />
        ))}
      </div>
    </Card>
  );
}
